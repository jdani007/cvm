# Cloud Volume Mapper (CVM)

Determine Google Cloud Storage bucket size per volume usage for Netapp Cloud Backup and Cloud Tiering.

## Usage:
```
Usage of ./cvm:
  -cluster string
        Enter cluster hostname or ip
  -export string
        Export .csv file. Enter 'local' or 'cloud' (default "none")
  -service string
        Enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service (default "backup")
```
<br>


## Minimum permissions

Netapp ONTAP user:

&nbsp; Role: **READONLY**

&nbsp; Mode: **HTTP**

Google Cloud Platorm:

&nbsp; Permissions: **storage.objects.list**

&nbsp; Permissions: **storage.objects.create** for cloud upload

<br>

## Environment Configuration

Environment variable is set via a file named ".env" in current folder.

```
netapp_auth="dXNlcm5hbWU6cGFzc3dvcmQ="
```

Authentication must be set with the user name and password encoded as a base64 string. For example:

```
$ echo -n "username:password" | base64
dXNlcm5hbWU6cGFzc3dvcmQ=
```

<br>

## Console output:
```
./cvm -cluster 192.168.0.1

Volume Size for Backup:

   Size     Volume Name             UUID                                  
   -----    ------------            -----                                 
1  1.71GB  temp_delete_me_too      8d8d9775-d79c-4fd2-a87d-674df87ec95f  
2  15.4GB  temp_delete_me          df4348fb-d8a9-4fc1-bf93-a8f8e66ef007  
3  1.71GB  temp_delete_me_as_well  bb1910c1-310c-41cb-8ce0-889cd108187a
```
<br>

Runs silent and creates a .csv file on the local filesystem.
```
./cvm -cluster 192.168.0.1 -export local
```
<br>

Runs silent and creates a .csv file on the local filesystem and uploads a copy to the cloud storage bucket.
```
./cvm -cluster 192.168.0.1 -export cloud
```


Cloud upload creates a 'report' subfolder in the Cloud Storage bucket of the corresponding service (backup | tiering).

<br>

### Typical buckets names:

Netapp Cloud Backup: **netapp-backup-\<random string>**

Netapp Cloud Tiering: **fabric-pool-\<random uuid>**