package dynamo

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
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
			{
				AttributeName: aws.String("domain_name"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("user_id"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("domain_name"),
				KeyType:       types.KeyTypeRange,
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

func (d *Storage) GetUserDomains(userId string) (map[string]models.DomainInformation, error) {
	var err error
	var response *dynamodb.QueryOutput
	var domainInfo []models.DomainInformation
	keyEx := expression.Key("user_id").Equal(expression.Value(userId))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, err
	} else {
		response, err = d.Client.Query(context.TODO(), &dynamodb.QueryInput{
			TableName:                 aws.String(d.TableName),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			KeyConditionExpression:    expr.KeyCondition(),
		})
		if err != nil {
			return nil, err
		} else {
			err = attributevalue.UnmarshalListOfMaps(response.Items, &domainInfo)
			if err != nil {
				return nil, err
			}
		}
	}
	retMap := make(map[string]models.DomainInformation)
	for _, domain := range domainInfo {
		retMap[domain.DomainName] = domain
	}

	return retMap, nil
}

func (d *Storage) GetDomainByUser(userId string, domain string) (models.DomainInformation, error) {
	domainInfo := models.DomainInformation{
		DomainName: domain,
		UserId:     userId,
	}

	key, err := domainInfo.GetKey()
	if err != nil {
		return models.DomainInformation{}, err
	}

	response, err := d.Client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		Key: key, TableName: aws.String(d.TableName),
	})
	if err != nil {
		return models.DomainInformation{}, err
	} else {
		err = attributevalue.UnmarshalMap(response.Item, &domainInfo)
		if err != nil {
			return models.DomainInformation{}, err
		}
	}
	return domainInfo, err
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

func (d *Storage) DeleteDomain(userId string, domain string) error {
	domainInfo := models.DomainInformation{
		DomainName: domain,
		UserId:     userId,
	}

	key, err := domainInfo.GetKey()
	if err != nil {
		return err
	}
	_, err = d.Client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(d.TableName), Key: key,
	})
	if err != nil {
		return err
	}
	return nil
}

func (d *Storage) GetAllRecords() ([]models.DomainInformation, error) {
	p := dynamodb.NewScanPaginator(d.Client, &dynamodb.ScanInput{TableName: &d.TableName})

	var domainInfos []models.DomainInformation
	for p.HasMorePages() {
		out, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}

		var pItems []models.DomainInformation
		err = attributevalue.UnmarshalListOfMaps(out.Items, &pItems)
		if err != nil {
			return nil, err
		}

		domainInfos = append(domainInfos, pItems...)
	}
	return domainInfos, nil
}
