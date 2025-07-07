package database

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Ensure dynamoAccessManager implements AccessManager
var _ AccessManager = (*dynamoAccessManager)(nil)

type dynamoAccessManager struct {
	client    *dynamodb.Client
	tableName string
}

// Initialize DynamoDB client
func NewDynamoAccessManager(ctx context.Context, tableName string) (AccessManager, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)
	return &dynamoAccessManager{
		client:    client,
		tableName: tableName,
	}, nil
}

// batchWrite to DynamoDB
func (r *dynamoAccessManager) batchWrite(ctx context.Context, writeRequests []types.WriteRequest) error {
	_, err := r.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			r.tableName: writeRequests,
		},
	})
	return err
}

// PutCredential inserts or updates a credential in the DynamoDB table
func (r *dynamoAccessManager) PutCredential(ctx context.Context, cred Credential) error {
	_, err := r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"Code":        &types.AttributeValueMemberS{Value: cred.Code},
			"Username":    &types.AttributeValueMemberS{Value: cred.Username},
			"AccessGroup": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", cred.AccessGroup)},
			"LockedOut":   &types.AttributeValueMemberBOOL{Value: cred.LockedOut},
		},
	})
	return err
}

// PutCredentials inserts with batch write
func (r *dynamoAccessManager) PutCredentials(ctx context.Context, creds []Credential) error {
	const batchSize = 25 // DynamoDB batch write size
	var writeRequests []types.WriteRequest

	for _, cred := range creds {
		item := map[string]types.AttributeValue{
			"Code":        &types.AttributeValueMemberS{Value: cred.Code},
			"Username":    &types.AttributeValueMemberS{Value: cred.Username},
			"AccessGroup": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", cred.AccessGroup)},
			"LockedOut":   &types.AttributeValueMemberBOOL{Value: cred.LockedOut},
		}

		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: item},
		})

		// If we reach batch size, send the batch write request
		if len(writeRequests) == batchSize {
			err := r.batchWrite(ctx, writeRequests)
			if err != nil {
				return err
			}
			writeRequests = nil // Reset
		}
	}

	if len(writeRequests) > 0 {
		if err := r.batchWrite(ctx, writeRequests); err != nil {
			return err
		}
	}

	return nil
}

// GetCredential retrieves a credential by its code
func (r *dynamoAccessManager) GetCredential(ctx context.Context, code string) (*Credential, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"Code": &types.AttributeValueMemberS{Value: code},
		},
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, fmt.Errorf("credential not found")
	}

	accessGroup, err := strconv.Atoi(out.Item["AccessGroup"].(*types.AttributeValueMemberN).Value)
	if err != nil {
		return nil, err
	}

	return &Credential{
		Code:        code,
		Username:    out.Item["Username"].(*types.AttributeValueMemberS).Value,
		AccessGroup: accessGroup,
		LockedOut:   out.Item["LockedOut"].(*types.AttributeValueMemberBOOL).Value,
	}, nil
}

// GetCredentials retrieves all credentials from the DynamoDB table.
func (r *dynamoAccessManager) GetCredentials(ctx context.Context) ([]Credential, error) {
	var credentials []Credential
	input := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	}

	result, err := r.client.Scan(ctx, input)
	if err != nil {
		return nil, err
	}

	for _, item := range result.Items {
		accessGroup, err := strconv.Atoi(item["AccessGroup"].(*types.AttributeValueMemberN).Value)
		if err != nil {
			return nil, err
		}

		cred := Credential{
			Code:        item["Code"].(*types.AttributeValueMemberS).Value,
			Username:    item["Username"].(*types.AttributeValueMemberS).Value,
			AccessGroup: accessGroup,
			LockedOut:   item["LockedOut"].(*types.AttributeValueMemberBOOL).Value,
		}
		credentials = append(credentials, cred)
	}

	return credentials, nil
}

// DeleteCredential deletes a credential by its code
func (r *dynamoAccessManager) DeleteCredential(ctx context.Context, code string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"Code": &types.AttributeValueMemberS{Value: code},
		},
	})
	return err
}

// DeleteCredentials deletes multiple credentials by their codes
func (r *dynamoAccessManager) DeleteCredentials(ctx context.Context, codes []string) error {
	if len(codes) == 0 {
		return nil // Nothing to delete
	}

	const batchSize = 25
	var writeRequests []types.WriteRequest

	for _, code := range codes {
		writeRequests = append(writeRequests, types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: map[string]types.AttributeValue{
					"Code": &types.AttributeValueMemberS{Value: code},
				},
			},
		})

		if len(writeRequests) == batchSize {
			if err := r.batchWrite(ctx, writeRequests); err != nil {
				return err
			}
			writeRequests = nil // Reset for the next batch
		}
	}

	// Final batch
	if len(writeRequests) > 0 {
		if err := r.batchWrite(ctx, writeRequests); err != nil {
			return err
		}
	}

	return nil
}

// PutAccessTime adds or updates AccessTime for a group
func (r *dynamoAccessManager) PutAccessTime(ctx context.Context, at AccessTime) error {
	_, err := r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"AccessGroup":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", at.AccessGroup)},
			"StartTime":    &types.AttributeValueMemberS{Value: at.StartTime.Format("15:04:05")},
			"EndTime":      &types.AttributeValueMemberS{Value: at.EndTime.Format("15:04:05")},
			"StartWeekday": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", at.StartWeekday)},
			"EndWeekday":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", at.EndWeekday)},
		},
	})
	return err
}

// GetAccessTime retrieves AccessTime for a specific access group
func (r *dynamoAccessManager) GetAccessTime(ctx context.Context, accessGroup int) (*AccessTime, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"AccessGroup": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", accessGroup)},
		},
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, fmt.Errorf("access time not found for group id <%d>", accessGroup)
	}

	startTimeStr := out.Item["StartTime"].(*types.AttributeValueMemberS).Value
	endTimeStr := out.Item["EndTime"].(*types.AttributeValueMemberS).Value

	startTime, err := time.ParseInLocation("15:04:05", startTimeStr, time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid start time format: %v", err)
	}

	endTime, err := time.ParseInLocation("15:04:05", endTimeStr, time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid end time format: %v", err)
	}

	startWeekday, err := strconv.Atoi(out.Item["StartWeekday"].(*types.AttributeValueMemberN).Value)
	if err != nil {
		return nil, fmt.Errorf("invalid start weekday: %v", err)
	}

	endWeekday, err := strconv.Atoi(out.Item["EndWeekday"].(*types.AttributeValueMemberN).Value)
	if err != nil {
		return nil, fmt.Errorf("invalid end weekday: %v", err)
	}

	return &AccessTime{
		AccessGroup:  accessGroup,
		StartTime:    startTime,
		EndTime:      endTime,
		StartWeekday: time.Weekday(startWeekday),
		EndWeekday:   time.Weekday(endWeekday),
	}, nil
}

// DeleteAccessTime removes AccessTime for a group
func (r *dynamoAccessManager) DeleteAccessTime(ctx context.Context, accessGroup int) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"AccessGroup": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", accessGroup)},
		},
	})
	return err
}
