package aws

type resourceMetadata struct {
	AWSCredentials awsCredentials
}
type awsCredentials struct {
	AccessKeyID     string
	AccessKeySecret string
}
