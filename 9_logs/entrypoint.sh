#!/bin/sh

# Delay for 10 seconds
echo "waiting logstash to start properly and connected to elasticsearch ..."
echo "please wait 50 seconds ..."
sleep 50

# Execute the main command
exec "$@"
