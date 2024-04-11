# mapstorage

Determine Google cloud storage per volume usage for Netapp Cloud Backup

still work in progress for tiering service

output

usage:
```bash
Usage of ./mapstorage:
  -cluster string
        enter cluster hostname or ip
  -export
        export to csv file
  -service string
        enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service (default "backup")
```

output
```bash
./mapstorage -cluster 192.168.0.1

Volume Size for Backup:

Size   Volume Name            UUID                                 
-----  ------------           -----                                
1.7GB  temp_delete_me_too     8d8d9775-d79c-4fd2-a87d-674df87ec95f 
15.4GB temp_delete_me         df4348fb-d8a9-4fc1-bf93-a8f8e66ef007 
1.7GB  temp_delete_me_as_well bb1910c1-310c-41cb-8ce0-889cd108187a 
```