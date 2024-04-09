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

7.7GB	temp_delete_me_too
15.4GB	temp_delete_me
1.7GB	temp_delete_me_as_well
```