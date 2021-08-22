## ARGUMENTS: MQTT Server, REST Server, Devices
docker-compose.yml:
    args
    - mqtt_uri=mqtt://pi:woofaa@172.17.0.1:1883
    - rest_uri=http://172.17.0.1/api/iaq/createrecord

*Restart after modification

## BUILD
docker-compose build

## RUN
For Debug:
docker-compose -f docker-compose.yml up -it 

For Deploy
docker-compose -f docker-compose.yml up -d

## STOP
docker-compose stop

## REMOVE and DELETE volumes
docker-compose down -v

## Logging
e.g. docker-compose logs -f > dockerlog.txt


## MQTT
Publish:
mosquitto_pub -d -h iaq.creaxtive.com -p 1883 -u pi -P woofaa -t 'GASDATA/8caab58daa99' -m '{\"mac\":\"8caab58daa99\", \"CO2\":678, \"temp\":28.5, \"hum\":65.3}'
mosquitto_pub -d -h localhost -p 1883 -u pi -P woofaa -t 'UVSTATUS/CURRENTLIFETIME/8caab58daa99' -m '{\"currentlifetime\":165}'
mosquitto_pub -d -h localhost -p 1883 -u pi -P woofaa -t 'UVSTATUS/SENDVALUE/8caab58daa99' -m '{\"uvstatus\":165}'

Subscrib:
mosquitto_sub -d -h iaq.creaxtive.com -p 1883  -u pi -P woofaa -t "#"
