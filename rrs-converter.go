package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"os/user"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)
type Attrs struct {
	Region     string
	Bucket     string
	Config     string
	Section    string
	Concurancy int
}

func getArgs() *Attrs {
	var region, config string
	regionPtr := flag.String("region", "", "Defines region")
	bucketPtr := flag.String("bucket", "", "Defines bucket. default = empty")
	configPtr := flag.String("creds", "", "Allow changing AWS account")
	sectionPtr := flag.String("section", "default", "Which part of AWS credentials to use")
	concurancyPtr := flag.Int("concurancy", 10, "Set up maximum concurancy for this task. Default is 10")
	flag.Parse()
	if *bucketPtr == "" {
		fmt.Println("You haven't define bucket! Please, do it with -bucket= ")
		os.Exit(1)
	}
	if *regionPtr == "" {
		region = "us-east-1"
		fmt.Println("You haven't specified region. Default region will be us-east-1\n")
	} else {
		region = *regionPtr
	}
	if *configPtr == "" {
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}
		config = usr.HomeDir + "/.aws/credentials"
	} else {
		config = *configPtr
	}
	attrs := Attrs {
	Region:     region,
	Bucket:     *bucketPtr,
	Config:     config,
	Section:    *sectionPtr,
	Concurancy: *concurancyPtr,
	}
	return &attrs
}

func copy(attrs, object string) {

}

func main() {
	attrs := getArgs()
	creds := credentials.NewSharedCredentials(attrs.Config, attrs.Section)
	_, err := creds.Get()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	svc := s3.New(session.New(), &aws.Config{
		Region:      aws.String(attrs.Region),
		Credentials: creds,
	})
	params := &s3.ListObjectsInput{
		Bucket: aws.String(attrs.Bucket),
	}
	resp, _ := svc.ListObjects(params)
	fmt.Println("Found ", len(resp.Contents), " objects.\n Processing... It could take a while...")

	throttle := make(chan int, attrs.Concurancy)
  var wg sync.WaitGroup
	for _, key := range resp.Contents {
		if *key.StorageClass != "REDUCED_REDUNDANCY" {
			throttle <- 1
			wg.Add(1)
			go func() {
				defer wg.Done()
				copyParams := &s3.CopyObjectInput{
				  Bucket: aws.String(attrs.Bucket),
				  CopySource: aws.String(attrs.Bucket + "/" + *key.Key),
				  Key: aws.String(*key.Key),
					StorageClass: aws.String("REDUCED_REDUNDANCY"),
			  }
				_, err := svc.CopyObject(copyParams)
				if err != nil {
					panic(err)
				}
				<-throttle
			}()
			wg.Wait()
		}
	}
	for i := 0; i < cap(throttle); i++ {
		throttle <- 1
	}
}
