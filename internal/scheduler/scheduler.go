package scheduler

import (
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/purell"
	"github.com/darkspot-org/bathyscaphe/internal/cache"
	configapi "github.com/darkspot-org/bathyscaphe/internal/configapi/client"
	"github.com/darkspot-org/bathyscaphe/internal/constraint"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"mvdan.cc/xurls/v2"
)

var (
	errNotOnionHostname    = errors.New("hostname is not .onion")
	errProtocolNotAllowed  = errors.New("protocol is not allowed")
	errExtensionNotAllowed = errors.New("extension is not allowed")
	errHostnameNotAllowed  = errors.New("hostname is not allowed")
	errAlreadyScheduled    = errors.New("URL is already scheduled")
)

// State represent the application state
type State struct {
	configClient configapi.Client
	urlCache     cache.Cache
}

// Name return the process name
func (state *State) Name() string {
	return "scheduler"
}

// Description return the process description
func (state *State) Description() string {
	return `
The scheduling component. It extracts URLs from crawled resources
and apply a predicate to determinate if the URL is eligible
for crawling. If it is, it will publish a event and update the
scheduling cache.
This component consumes the 'resource.new' event and produces
the 'url.new' event.`
}

// Features return the process features
func (state *State) Features() []process.Feature {
	return []process.Feature{process.EventFeature, process.ConfigFeature, process.CacheFeature}
}

// CustomFlags return process custom flags
func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{}
}

// Initialize the process
func (state *State) Initialize(provider process.Provider) error {
	keys := []string{configapi.AllowedMimeTypesKey, configapi.ForbiddenHostnamesKey, configapi.RefreshDelayKey}
	configClient, err := provider.ConfigClient(keys)
	if err != nil {
		return err
	}
	state.configClient = configClient

	urlCache, err := provider.Cache("url")
	if err != nil {
		return err
	}
	state.urlCache = urlCache

	return nil
}

// Subscribers return the process subscribers
func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{
		{Exchange: event.NewResourceExchange, Queue: "schedulingQueue", Handler: state.handleNewResourceEvent},
	}
}

// HTTPHandler returns the HTTP API the process expose
func (state *State) HTTPHandler() http.Handler {
	return nil
}

func (state *State) handleNewResourceEvent(subscriber event.Subscriber, msg event.RawMessage) error {
	var evt event.NewResourceEvent
	if err := subscriber.Read(&msg, &evt); err != nil {
		return err
	}

	log.Trace().Str("url", evt.URL).Msg("Processing new resource")

	urls, err := extractURLS(&evt)
	if err != nil {
		return fmt.Errorf("error while extracting URLs")
	}

	// We are working using URL hash to reduce memory consumption.
	// See: https://github.com/darkspot-org/bathyscaphe/issues/130
	var urlHashes []string
	for _, u := range urls {
		c := fnv.New64()
		if _, err := c.Write([]byte(u)); err != nil {
			return fmt.Errorf("error while computing url hash: %s", err)
		}

		urlHashes = append(urlHashes, strconv.FormatUint(c.Sum64(), 10))
	}

	// Load values in batch
	urlCache, err := state.urlCache.GetManyInt64(urlHashes)
	if err != nil {
		return err
	}

	for _, u := range urls {
		if err := state.processURL(u, subscriber, urlCache); err != nil {
			log.Err(err).Msg("error while processing URL")
		}
	}

	// Update URL cache
	delay, err := state.configClient.GetRefreshDelay()
	if err != nil {
		return err
	}

	// Update values in batch
	if err := state.urlCache.SetManyInt64(urlCache, delay.Delay); err != nil {
		return err
	}

	return nil
}

// // Old version
// func (state *State) processURL(rawURL string, pub event.Publisher, urlCache map[string]int64) error {
// 	u, err := url.Parse(rawURL)
// 	if err != nil {
// 		return fmt.Errorf("error while parsing URL: %s", err)
// 	}

// 	// Make sure URL is valid .onion
// 	if !strings.HasSuffix(u.Hostname(), ".onion") {
// 		return fmt.Errorf("%s %w", u.Host, errNotOnionHostname)
// 	}

// 	// Make sure protocol is not forbidden
// 	if !strings.HasPrefix(u.Scheme, "http") {
// 		return fmt.Errorf("%s %w", u, errProtocolNotAllowed)
// 	}

// 	// Make sure extension is allowed
// 	allowed := false
// 	if mimeTypes, err := state.configClient.GetAllowedMimeTypes(); err == nil {
// 		for _, mimeType := range mimeTypes {
// 			for _, ext := range mimeType.Extensions {
// 				if strings.HasSuffix(strings.ToLower(u.Path), "."+ext) {
// 					allowed = true
// 				}
// 			}
// 		}
// 	}

// 	// Handle case no extension present
// 	if !allowed {
// 		components := strings.Split(u.Path, "/")

// 		lastIdx := 0
// 		if size := len(components); size > 0 {
// 			lastIdx = size - 1
// 		}

// 		// generally no extension means text/* content-type
// 		if !strings.Contains(components[lastIdx], ".") {
// 			allowed = true
// 		}
// 	}

// 	if !allowed {
// 		return fmt.Errorf("%s %w", u, errExtensionNotAllowed)
// 	}

// 	// Make sure hostname is not forbidden
// 	if allowed, err := constraint.CheckHostnameAllowed(state.configClient, rawURL); err != nil {
// 		return err
// 	} else if !allowed {
// 		log.Debug().Str("url", rawURL).Msg("Skipping forbidden hostname")
// 		return fmt.Errorf("%s %w", u, errHostnameNotAllowed)
// 	}

// 	// Compute url hash
// 	c := fnv.New64()
// 	if _, err := c.Write([]byte(rawURL)); err != nil {
// 		return fmt.Errorf("error while computing url hash: %s", err)
// 	}
// 	urlHash := strconv.FormatUint(c.Sum64(), 10)

// 	// Check if URL should be scheduled
// 	if urlCache[urlHash] > 0 {
// 		return fmt.Errorf("%s %w", u, errAlreadyScheduled)
// 	}

// 	log.Debug().Stringer("url", u).Msg("URL should be scheduled")

// 	urlCache[urlHash]++

// 	if err := pub.PublishEvent(&event.NewURLEvent{URL: rawURL}); err != nil {
// 		return fmt.Errorf("error while publishing URL: %s", err)
// 	}

// 	return nil
// }

// HTW CHANGE
// Only schedule domain URLs
func (state *State) processURL(rawURL string, pub event.Publisher, urlCache map[string]int64) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("error while parsing URL: %s", err)
	}

	// Make sure URL is valid .onion
	if !strings.HasSuffix(u.Hostname(), ".onion") {
		return fmt.Errorf("%s %w", u.Host, errNotOnionHostname)
	}

	// Make sure protocol is not forbidden
	if !strings.HasPrefix(u.Scheme, "http") {
		return fmt.Errorf("%s %w", u, errProtocolNotAllowed)
	}

	// Make sure extension is allowed
	allowed := false
	if mimeTypes, err := state.configClient.GetAllowedMimeTypes(); err == nil {
		for _, mimeType := range mimeTypes {
			for _, ext := range mimeType.Extensions {
				if strings.HasSuffix(strings.ToLower(u.Path), "."+ext) {
					allowed = true
				}
			}
		}
	}

	// Handle case no extension present
	if !allowed {
		components := strings.Split(u.Path, "/")

		lastIdx := 0
		if size := len(components); size > 0 {
			lastIdx = size - 1
		}

		// generally no extension means text/* content-type
		if !strings.Contains(components[lastIdx], ".") {
			allowed = true
		}
	}

	if !allowed {
		return fmt.Errorf("%s %w", u, errExtensionNotAllowed)
	}

	// Make sure hostname is not forbidden
	if allowed, err := constraint.CheckHostnameAllowed(state.configClient, rawURL); err != nil {
		return err
	} else if !allowed {
		log.Debug().Str("url", rawURL).Msg("Skipping forbidden hostname")
		return fmt.Errorf("%s %w", u, errHostnameNotAllowed)
	}

	// Make sure hostname is not forbidden
	if allowed := isForbidden(u.Hostname()); !allowed {
		log.Debug().Str("url", rawURL).Msg("Skipping forbidden hostname")
		return fmt.Errorf("%s %w", u, errHostnameNotAllowed)
	}

	// Compute url hash
	c := fnv.New64()
	if _, err := c.Write([]byte(rawURL)); err != nil {
		return fmt.Errorf("error while computing url hash: %s", err)
	}
	urlHash := strconv.FormatUint(c.Sum64(), 10)

	// Check if URL should be scheduled
	if urlCache[urlHash] > 0 {
		return fmt.Errorf("%s %w", u, errAlreadyScheduled)
	}

	log.Debug().Stringer("url", u).Msg("URL should be scheduled")

	urlCache[urlHash]++

	if err := pub.PublishEvent(&event.NewURLEvent{URL: rawURL}); err != nil {
		return fmt.Errorf("error while publishing URL: %s", err)
	}

	return nil
}

// HTW CHANGE
// Only schedule domain URL
func extractURLS(msg *event.NewResourceEvent) ([]string, error) {
	// Extract & normalize URLs
	xu := xurls.Strict()
	urls := xu.FindAllString(msg.Body, -1)

	var normalizedURLS []string

	for _, u := range urls {
		normalizedURL, err := normalizeURL(u)
		if err != nil {
			continue
		}

		urlparsed, err := url.Parse(normalizedURL)

		domainURL := "http://" + urlparsed.Hostname()

		log.Debug().Str("url", domainURL).Msg("Domain URL: ")

		normalizedURLS = append(normalizedURLS, domainURL)
	}

	return normalizedURLS, nil
}

// // Old version
// func extractURLS(msg *event.NewResourceEvent) ([]string, error) {
// 	// Extract & normalize URLs
// 	xu := xurls.Strict()
// 	urls := xu.FindAllString(msg.Body, -1)

// 	var normalizedURLS []string

// 	for _, u := range urls {
// 		normalizedURL, err := normalizeURL(u)
// 		if err != nil {
// 			continue
// 		}

// 		normalizedURLS = append(normalizedURLS, normalizedURL)
// 	}

// 	return normalizedURLS, nil
// }

func normalizeURL(u string) (string, error) {
	normalizedURL, err := purell.NormalizeURLString(u, purell.FlagsUsuallySafeGreedy|
		purell.FlagRemoveDirectoryIndex|purell.FlagRemoveFragment|purell.FlagRemoveDuplicateSlashes)
	if err != nil {
		return "", fmt.Errorf("error while normalizing URL %s: %s", u, err)
	}

	return normalizedURL, nil
}

// Check if a string is in a list
func stringInArray(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Check if a hostname is forbidden
func isForbidden(url string) bool {
	forbiddenHostnames := []string{"www.facebookcorewwwi.onion",
		"facebookcorewwwi.onion",
		"m.facebookcorewwwi.onion",
		"ar-ar.facebookcorewwwi.onion",
		"bg-bg.facebookcorewwwi.onion",
		"bs-ba.facebookcorewwwi.onion",
		"ca-es.facebookcorewwwi.onion",
		"da-dk.facebookcorewwwi.onion",
		"el-gr.facebookcorewwwi.onion",
		"es-la.facebookcorewwwi.onion",
		"es-es.facebookcorewwwi.onion",
		"fa-ir.facebookcorewwwi.onion",
		"fi-fi.facebookcorewwwi.onion",
		"fr-fr.facebookcorewwwi.onion",
		"fr-ca.facebookcorewwwi.onion",
		"hi-in.facebookcorewwwi.onion",
		"hr-hr.facebookcorewwwi.onion",
		"id-id.facebookcorewwwi.onion",
		"it-it.facebookcorewwwi.onion",
		"ko-kr.facebookcorewwwi.onion",
		"mk-mk.facebookcorewwwi.onion",
		"ms-my.facebookcorewwwi.onion",
		"pl-pl.facebookcorewwwi.onion",
		"pt-br.facebookcorewwwi.onion",
		"pt-pt.facebookcorewwwi.onion",
		"ro-ro.facebookcorewwwi.onion",
		"sl-si.facebookcorewwwi.onion",
		"sr-rs.facebookcorewwwi.onion",
		"th-th.facebookcorewwwi.onion",
		"vi-vn.facebookcorewwwi.onion",
		"de-de.facebookcorewwwi.onion",
		"zh-cn.facebookcorewwwi.onion",
		"ja-jp.facebookcorewwwi.onion",
		"pay.facebookcorewwwi.onion",
		"portal.facebookcorewwwi.onion",
		"l.facebookcorewwwi.onion",
		"developers.facebookcorewwwi.onion",
		"pixel.facebookcorewwwi.onion",
		"static.xx.fbcdn23dssr3jqnq.onion"}
	allowed := !stringInArray(url, forbiddenHostnames)
	if allowed {
		log.Debug().Str("url", url).Msg("URL is allowed: ")
	} else {
		log.Debug().Str("url", url).Msg("URL is forbidden: ")
	}
	return allowed
}
