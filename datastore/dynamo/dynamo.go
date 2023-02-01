package dynamo

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/edwinavalos/dns-verifier/datastore"
	"github.com/edwinavalos/dns-verifier/models"
	"time"
)

type Storage struct {
	TableName string
	Client    *dynamodb.Client
}

func NewStorage() (datastore.Datastore, error) {
	var conf aws.Config
	var err error
	if datastore.Config.DB.IsLocal {
		conf, err = aws_config.LoadDefaultConfig(context.TODO(),
			aws_config.WithRegion("us-east-1"),
			aws_config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: "http://localhost:8000"}, nil
				})),
			aws_config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID: "dummy", SecretAccessKey: "dummy", SessionToken: "dummy",
					Source: "Hard-coded credentials; values are irrelevant for local DynamoDB",
				},
			}),
		)
		if err != nil {
			return &Storage{}, fmt.Errorf("unable to create local client: %w", err)
		}
	} else {
		conf, err = aws_config.LoadDefaultConfig(context.TODO(), func(o *aws_config.LoadOptions) error {
			o.Region = "us-east-1"
			return nil
		})
		if err != nil {
			return &Storage{}, fmt.Errorf("unable to create remote dynamod client: %w", err)
		}
	}

	dynamodbClient := dynamodb.NewFromConfig(conf)
	return &Storage{
		TableName: datastore.Config.DB.TableName,
		Client:    dynamodbClient,
	}, nil
}

func (d *Storage) GetTableName() string {
	return d.TableName
}

func (d *Storage) DropTable() error {
	d.Client.DeleteTable(context.TODO(), &dynamodb.DeleteTableInput{
		TableName: &d.TableName,
	})
	return nil
}

func (d *Storage) tableExists() (bool, error) {
	exists := true
	_, err := d.Client.DescribeTable(
		context.TODO(), &dynamodb.DescribeTableInput{TableName: aws.String(d.TableName)},
	)
	if err != nil {
		var notFoundEx *types.ResourceNotFoundException
		if errors.As(err, &notFoundEx) {
			datastore.Log.Infof("Table %v does not exist", d.TableName)
			err = nil
		} else {
			datastore.Log.Infof("Couldn't determine existence of table %v %v", d.TableName, err)
		}
		exists = false
	}
	return exists, err
}

func (d *Storage) Initialize() error {
	exists, err := d.tableExists()
	if exists {
		datastore.Log.Infof("table: %s already exists, not creating", d.TableName)
		return nil
	}
	_, err = d.Client.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("user_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("user_id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		TableName: aws.String(d.TableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	})
	if err != nil {
		return fmt.Errorf("unable to create table: %w", err)
	} else {
		waiter := dynamodb.NewTableExistsWaiter(d.Client)
		err = waiter.Wait(context.TODO(), &dynamodb.DescribeTableInput{
			TableName: aws.String(d.TableName)}, 5*time.Minute)
		if err != nil {
			return fmt.Errorf("wait for table exists failed: %w", err)
		}
	}
	return err
}

func (d *Storage) GetUserDomains(userId string) map[string]models.DomainInformation {
	
	return map[string]models.DomainInformation{}
}

func (d *Storage) GetDomainByUser(userId string, domain string) models.DomainInformation {
	return models.DomainInformation{}
}

func (d *Storage) PutDomainInfo(info models.DomainInformation) error {
	item, err := attributevalue.MarshalMap(info)
	if err != nil {
		panic(err)
	}
	_, err = d.Client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(d.TableName), Item: item,
	})
	if err != nil {
		datastore.Log.Printf("Couldn't add item to table. Here's why: %v", err)
	}
	return err
}

func (d *Storage) DeleteDomain(userId string, domain string) {}

func (d *Storage) DeleteUser(userId string) {}
