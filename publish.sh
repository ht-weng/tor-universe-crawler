while read -r line; do 
    echo $line | rabbitmqadmin -H 192.168.1.120 -P 15003 -u guest -p guest publish exchange=amq.default routing_key=crawlingQueue; 
done < seed-urls.txt