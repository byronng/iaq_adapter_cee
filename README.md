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

