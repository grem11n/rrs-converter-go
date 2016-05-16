package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Attrs store user-defined parameters
type Attrs struct {
	Region      string
	Bucket      string
	Config      string
	Section     string
	Concurrency int
}

// Get user-defined parameters from CLI. Also, this function provides
// some verbose output, for example, if bucket was not specified
func getArgs() *Attrs {
	var region, config string
	regionPtr := flag.String("region", "", "Defines region")
	bucketPtr := flag.String("bucket", "", "Defines bucket. default = empty")
	configPtr := flag.String("config", "", "Allow changing AWS account")
	sectionPtr := flag.String("section", "default", "Which part of AWS credentials to use")
	concurrencyPtr := flag.Int("maxcon", 10, "Set up maximum concurrency for this task. Default is 10")
	flag.Parse()
	if *bucketPtr == "" {
		fmt.Println("You haven't define bucket! Please, do it with -bucket= ")
		os.Exit(1)
	}
	if *regionPtr == "" {
		region = "us-east-1"
		fmt.Println("You haven't specified region. Default region will be us-east-1")
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
	attrs := Attrs{
		Region:      region,
		Bucket:      *bucketPtr,
		Config:      config,
		Section:     *sectionPtr,
		Concurrency: *concurrencyPtr,
	}
	return &attrs
}

func main() {
	attrs := getArgs()
	creds := credentials.NewSharedCredentials(attrs.Config, attrs.Section)
	_, err := creds.Get()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Create new connection to S3
	svc := s3.New(session.New(), &aws.Config{
		Region:      aws.String(attrs.Region),
		Credentials: creds,
	})
	params := &s3.ListObjectsInput{
		Bucket: aws.String(attrs.Bucket),
	}
	resp, _ := svc.ListObjects(params)
	fmt.Print(len(resp.Contents), " objects in the bucket.\n Processing... It could take a while...")

	// This is used to limit simultaneous goroutines
	throttle := make(chan int, attrs.Concurrency)
	var wg sync.WaitGroup

	// Loop trough the objects in the bucket and create a copy
	// of each object with the REDUCED_REDUNDANCY storage class
	for _, key := range resp.Contents {
		if *key.StorageClass != "REDUCED_REDUNDANCY" {
			throttle <- 1
			wg.Add(1)
			go func() {
				defer wg.Done()
				copyParams := &s3.CopyObjectInput{
					Bucket:       aws.String(attrs.Bucket),
					CopySource:   aws.String(attrs.Bucket + "/" + *key.Key),
					Key:          aws.String(*key.Key),
					StorageClass: aws.String("REDUCED_REDUNDANCY"),
				}
				_, err := svc.CopyObject(copyParams)
				if err != nil {
					panic(err)
				}
				fmt.Print(".")
				<-throttle
			}()
			wg.Wait()
		}
	}

	// Fill the channel to be sure, that all goroutines finished
	for i := 0; i < cap(throttle); i++ {
		throttle <- 1
	}
	fmt.Println("\nConversion done!")
}
