package storage

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/edwinavalos/common/config"
	"github.com/edwinavalos/common/datastore/dynamo"
	"github.com/edwinavalos/common/logger"
	"github.com/edwinavalos/common/models"
	"github.com/google/uuid"
	"time"
)

type VerifierDataStore struct {
	*dynamo.Storage
}

func NewDataStore(conf *config.Config) (*VerifierDataStore, error) {
	storage, err := dynamo.New(conf)
	err = storage.NewLockTable()
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	exists, err := storage.TableExists()
	if exists {
		logger.Info("table: %s already exists, not creating", storage.TableName)
		return &VerifierDataStore{
			Storage: storage,
		}, nil
	}
	createTableInput := &dynamodb.CreateTableInput{
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
		TableName: aws.String(storage.TableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	_, err = storage.Client.CreateTable(context.TODO(), createTableInput)
	if err != nil {
		return nil, fmt.Errorf("unable to create table: %w", err)
	}
	waiter := dynamodb.NewTableExistsWaiter(&storage.Client)
	err = waiter.Wait(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(storage.TableName)}, 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("wait for table exists failed: %w", err)
	}

	return &VerifierDataStore{Storage: storage}, nil
}

//	userInfo := models.User{ID: userID}
//
//	key, err := userInfo.GetKey()
//	if err != nil {
//		return models.User{}, err
//	}

func (v *VerifierDataStore) GetUser(ctx context.Context, userID uuid.UUID) (models.User, error) {
	userInfo := models.User{ID: userID.String()}
	key, err := userInfo.GetKey()
	if err != nil {
		return models.User{}, err
	}
	output, err := v.Storage.GetByID(ctx, key)
	if err != nil {
		return userInfo, err
	}

	err = attributevalue.UnmarshalMap(output.Item, &userInfo)
	if err != nil {
		return models.User{}, err
	}

	return userInfo, nil
}

func (v *VerifierDataStore) GetDomainByUser(ctx context.Context, userID uuid.UUID, domain string) (models.DomainInformation, error) {
	userInfo, err := v.GetUser(ctx, userID)
	if err != nil {
		return models.DomainInformation{}, err
	}

	for _, domainInfo := range userInfo.Domains {
		if domainInfo.DomainName == domain {
			return domainInfo, nil
		}
	}

	return models.DomainInformation{}, fmt.Errorf("user: %s does not have a domain entry for: %s", userID, domain)
}

func (v *VerifierDataStore) PutDomainInfo(ctx context.Context, domainInfo models.DomainInformation) error {
	userInfo, err := v.GetUser(ctx, domainInfo.UserID)
	if err != nil {
		return err
	}

	if userInfo.Domains == nil {
		userInfo.Domains = make(map[string]models.DomainInformation)
	}
	userInfo.Domains[domainInfo.DomainName] = domainInfo
	item, err := attributevalue.MarshalMap(userInfo)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, "lock_key", userInfo.ID)
	err = v.Storage.PutItem(ctx, item)
	if err != nil {
		return err
	}
	return nil
}

func (v *VerifierDataStore) GetUserDomains(ctx context.Context, userID uuid.UUID) (map[string]models.DomainInformation, error) {
	userInfo, err := v.GetUser(ctx, userID)
	if err != nil {
		return map[string]models.DomainInformation{}, err
	}

	return userInfo.Domains, nil
}

func (v *VerifierDataStore) DeleteDomain(ctx context.Context, userID uuid.UUID, domainName string) error {
	userInfo, err := v.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	delete(userInfo.Domains, domainName)
	return nil
}

func (v *VerifierDataStore) GetAllRecords(ctx context.Context) ([]models.User, error) {
	var retRecords []models.User
	records, err := v.Storage.GetAllRecords(ctx)
	if err != nil {
		return retRecords, err
	}

	for _, record := range records {
		for _, item := range record.Items {
			user := models.User{}
			err = attributevalue.UnmarshalMap(item, &user)
			if err != nil {
				return []models.User{}, err
			}
			retRecords = append(retRecords, user)
		}
	}

	return retRecords, nil
}
