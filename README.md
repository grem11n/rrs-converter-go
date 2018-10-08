# RRS-converter Go

This script allows you to simply convert storage class of the AWS S3 objects. More information
about AWS S3 storage classes could be found [here](http://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html).

Script leverages
[AWS-sdk](https://github.com/aws/aws-sdk-go)
and some magic of goroutines to convert all objects in S3 bucket to the storage class you specify. If you want to convert only some of them, feel free to modify this script or use
[AWS CLI](http://www.developmentshack.com/amazon-s3-command-line-optionstipstricks/42).

### Build and install
##### Dependencies:
- [aws-sdk-go](https://github.com/aws/aws-sdk-go)
- [cheggaaa/pb](https://github.com/cheggaaa/pb)
- [fatih/color](https://github.com/fatih/color)


##### Installation:
Code is compatible with Go 1.11 and uses its [`Go Modules`](https://github.com/golang/go/wiki/Modules) Installation:

[Install Go 1.11](https://golang.org/doc/install) and do:
```
$ git clone https://github.com/grem11n/rrs-converter-go.git
$ go build rrs-converter.go
```

### Usage
#### Required Parameters

- You must specify which bucket to convert with `-bucket` flag. This is the only parameter, which is strictly required

##### Optional Parameters:

- `-config` - custom config location. I've never tried it with relative path. It could probably works, who knows. If not declared, AWS-sdk use your `~/.aws/credentials` file by default.
- `-region` - bucket location. Script will use `us-east-1` by default.
- `-section` specifies which section of your AWS configuration file to use. If not specified, use "[default]".
- `-maxcon` - number of maximum concurrent goroutines. 10 by default.
- `-type` - specifies target AWS storage class, `STANDARD` by default.

### Sample Usage

```
rrs-converter -bucket=my-bucket -config="/home/user/.aws/credentials" -region=eu-west-1 -section=test -maxcon=5 -type="reduced_redundancy"
```
