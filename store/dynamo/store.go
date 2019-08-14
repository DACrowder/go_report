package dynamo

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go_report/domain"
	"go_report/failure"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type Store struct {
	db    *dynamodb.DynamoDB
	log   *log.Logger
	Table string
}

func New(sesh *session.Session, tableName string, logger *log.Logger) (s *Store) {
	s = new(Store)
	s.Table, s.db, s.log = tableName, dynamodb.New(sesh), logger
	return s
}

func (s *Store) NewEntry(r domain.Report) (rr domain.Receipt, err error) {
	buf := new(bytes.Buffer)
	if err = json.NewEncoder(buf).Encode(r); err != nil {
		s.log.Fatal(err)
		return domain.Receipt{}, err
	}
	r.Key = getMD5HashString(buf.Bytes())
	av, err := dynamodbattribute.MarshalMap(r)
	if err != nil {
		return domain.Receipt{}, errToFailure(err)
	}
	_, err = s.db.PutItem(&dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(s.Table),
	})
	if err != nil {
		return domain.Receipt{}, errToFailure(err)
	}
	return domain.Receipt{GID: r.GID, Key: r.Key}, nil
}

func (s *Store) Select(rr domain.Receipt) (*domain.Report, error) {
	av, err := dynamodbattribute.MarshalMap(rr)
	if err != nil {
		return nil, errToFailure(err)
	}
	fmt.Printf("%+v", av)
	res, err := s.db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.Table),
		Key:       av,
	})
	if err != nil {
		return nil, errToFailure(err)
	}
	rpt := new(domain.Report)
	if err = dynamodbattribute.UnmarshalMap(res.Item, rpt); err != nil {
		return nil, errToFailure(err)
	}
	return rpt, nil
}

func (s *Store) SelectAll() ([]domain.Report, error) {
	params := &dynamodb.ScanInput{
		TableName: aws.String(s.Table),
	}
	res, err := s.db.Scan(params)
	if err != nil {
		return nil, errToFailure(err)
	}
	return unmarshalListOfMapsResult(res)
}

func createGroupScanExpr(gid string) (expression.Expression, error) {
	expr, err := expression.NewBuilder().WithFilter(
		expression.Name("gid").Equal(expression.Value(gid)),
	).Build()
	if err != nil {
		return expression.Expression{}, errToFailure(err)
	}
	return expr, nil
}

func (s *Store) SelectGroup(gid string) ([]domain.Report, error) {
	expr, err := createGroupScanExpr(gid)
	if err != nil {
		return nil, err
	}
	res, err := s.db.Scan(&dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		TableName:                 aws.String(s.Table),
	})
	if err != nil {
		return nil, errToFailure(err)
	}
	return unmarshalListOfMapsResult(res)
}

func (s *Store) RemoveEntry(rr domain.Receipt) error {
	av, err := dynamodbattribute.MarshalMap(rr)
	if err != nil {
		return errToFailure(err)
	}
	_, err = s.db.DeleteItem(&dynamodb.DeleteItemInput{
		Key:       av,
		TableName: aws.String(s.Table),
	})
	if err != nil {
		return errToFailure(err)
	}
	return nil
}

func errToFailure(err error) *failure.RequestFailure {
	switch err.(type) {
	case *dynamodbattribute.InvalidMarshalError:
		return failure.New(err, http.StatusBadRequest, "")
	default:
		switch code := err.Error(); code {
		case dynamodb.ErrCodeIndexNotFoundException:
			return failure.New(err, http.StatusNotFound, code)
		}
		return failure.New(err, http.StatusInternalServerError, "")
	}
}

func getMD5HashString(text []byte) string {
	hasher := md5.New()
	hasher.Write(text)
	return hex.EncodeToString(hasher.Sum(nil))
}

func unmarshalListOfMapsResult(res *dynamodb.ScanOutput) ([]domain.Report, error) {
	rpts := make([]domain.Report, 0, 32)
	if err := dynamodbattribute.UnmarshalListOfMaps(res.Items, &rpts); err != nil {
		return nil, errToFailure(err)
	}
	return rpts, nil
}
