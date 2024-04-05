# mapstorage

Determine Google cloud storage per volume usage for Netapp Cloud Backup

still work in progress

output

usage:
```bash
Usage of ./mapstorage:
  -cluster string
        enter cluster hostname or ip
```

output
```bash
./mapstorage -cluster 192.168.0.1

1.7GiB sdf9637sdf77-4dfe5e-4406-a8b9-8cd2dfdb27ed temp_delete_me_too
7.2GiB e7239018-d23sdf6-4db1-93b8-0erwb2e7afscfdd temp_delete_me_as_well
15.4GiB df4348fb-d8fsa9-4fc1-bf93-a8fddafe66ef007f temp_delete_me
```