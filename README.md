# Bathyscaphe dark web crawler

![CI](https://github.com/darkspot-org/bathyscaphe/workflows/CI/badge.svg)

Bathyscaphe is a Go written, fast, highly configurable, cloud-native dark web crawler.

## How to start the crawler

### Without tor bridges

Execute the `./scripts/docker/start.sh` and wait for all containers to start.
You can start the crawler in detached mode by passing --detach to start.sh

Ensure that image `dperson/torproxy:latest` is used in `docker-compose.yml` in deployments/docker.

```yml
# torproxy:
#   image: torproxy:Dockerfile
#   logging:
#     driver: none
torproxy:
  image: dperson/torproxy:latest
  logging:
    driver: none
```

### With tor bridges

`cd build/tor-proxy/`. Then edit the torrc file to add tor bridges.

Tor bridges configurations can be found in `tor-browser_en-US/Browser/TorBrowser/Data/Tor/torrc`.

Execute `docker build -t "torproxy:Dockerfile" .` to build the image locally.

Then modify `docker-compose.yml` in `deployments/docker`.

```yml
# replace dperson/torproxy with torproxy built locally from niruix/tor
torproxy:
  image: torproxy:Dockerfile
  logging:
    driver: none
# torproxy:
#   image: dperson/torproxy:latest
#   logging:
#     driver: none
```

### Start the crawler

```sh
./scripts/docker/start.sh
```

### Note

- You can start the crawler in detached mode by passing --detach to start.sh.
- Ensure you have at least 3 GB of memory as the Elasticsearch stack docker will require 2 GB.

## How to store ElasticSearch data in a specific folder

Modify the `docker-compose.yml` file. Replace the named volume with path to the folder.

```yml
elasticsearch:
  image: elasticsearch:7.5.1
  logging:
    driver: none
  environment:
    - discovery.type=single-node
    - ES_JAVA_OPTS=-Xms2g -Xmx2g
  volumes:
    - /mnt/NAStor-universe/esdata:/usr/share/elasticsearch/data
```

## How to initiate the crawling process

One can use the RabbitMQ dashhboard available at [RabbitMQ](http://localhost:15003/), and publish a new JSON object in the **crawlingQueue**.

The object should look like this:

```json
{"url": "http://torlinkbgs6aabns.onion/"}
```

Multiple URLs can be published automatically using [rabbitmqadmin](https://www.rabbitmq.com/management-cli.html).

Go to `http://{hostname}:15672/cli/rabbitmqadmin` to download `rabbitmqadmin`.

Then `sudo chmod +x rabbitmqadmin`, `sudo cp rabbitmqadmin /usr/local/bin`.

Finally run `./publish.sh` to publish seed URLs.

## How to speed up crawling

If one want to speed up the crawling, he can scale the instance of crawling component in order to increase performance.  

This may be done by issuing the following command after the crawler is started:

```sh
./scripts/docker/start.sh --scale crawler=10 --scale indexer-es=2 --scale scheduler=4
```

## How to view results

### Using kibana

You can use the [Kibana dashboard](http://localhost:15004).  

You will need to create an index pattern named 'resources', and when it asks for the time field, choose 'time'.

## How to connect to docker containers

```sh
docker exec -it <docker container name> bash
```

## How to kill all docker containers

```sh
docker container kill $(docker ps -q)
```

## How to export data from ElasticSearch DB to a file

Install [elasticdump](https://github.com/elasticsearch-dump/elasticsearch-dump)

```sh
elasticdump --input=http://[elasticsearch-url]:9200/resources --output=[file_path]/universe.json --limit 500 --concurrency 20 --concurrencyInterval 1 --type=data --max-old-space-size=16384
```

```sh
elasticdump --input=http://172.18.0.3:9200/resources --output=/home/justin/Public/universe_data/universe-mar-26.json --limit 500 -concurrency 20 --concurrencyInterval 1 --type=data --max-old-space-size=16384
```

## How to build your own crawler

If you've made a change to one of the crawler component and wish to use the updated version when running start.sh you just need to issue the following command:

```sh
goreleaser --snapshot --skip-publish --rm-dist
```

This will rebuild all images using local changes. After that just run `start.sh` again to have the updated version
running.

Example:

## How to deal with Error (FORBIDDEN/12/index read-only / allow delete (api)])

```sh
PUT _settings
{
  "index": {
    "blocks": {
    "read_only_allow_delete": "false"
    }
  }
}
```

## How to analyse the universe

Run `universe-mining.ipynb` for general analysis and `classification.ipynb` for domain classification.

### Install dependencies using `conda`

```sh
conda install -c anaconda py-xgboost
```

### Build a Neural Network for classification

#### Download training dataset

First download the labelled darknet addresses provided in `DUTA_10K.xls` by [GVIS](http://gvis.unileon.es/dataset/duta-darknet-usage-text-addresses-10k/).

```sh
cd page-downloader/
python3 downloader.py
```

The downloaded webpages are in `data/universe-labelled`

POST http://172.23.0.3:9200/v1/resources/_delete_by_query
{
  "query": {
    "match": {
      "url":"http://torlinkbgs6aabns.onion"
    }
  }
}

POST /resources/_delete_by_query
{
  "query": {
    "match": {
      "url":"http://torlinkbgs6aabns.onion"
    }
  }
}

#### Classify darknet websites

All classifiers are in the `classification` folder.
