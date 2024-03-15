echo "start to clean all client"
pkill -9 ahluser
pkill -9 ahlclient
pkill -9 ahlattack
pkill -9 ahllatency

pkill -9 bysharduser
pkill -9 byshardclient
pkill -9 byshardlatency
pkill -9 byshardattack

pkill -9 haechiclient
pkill -9 haechilatency
pkill -9 haechiuser
pkill -9 haechiattack

pkill -9 haechisyncclient
pkill -9 haechisynclatency
pkill -9 haechisyncuser
pkill -9 haechisyncattack

echo "clean finished"