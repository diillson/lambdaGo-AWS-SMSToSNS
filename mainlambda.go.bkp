package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sirupsen/logrus"
)

type Response struct {
	StatusCode int    `json:"statusCode"`
	Body       string `json:"body"`
}

var log = logrus.New()

func init() {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetReportCaller(true)
}

func HandleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (Response, error) {
	lc, _ := lambdacontext.FromContext(ctx)
	awsRequestID := lc.AwsRequestID

	// Adicionar o nome da aplicação e o AWS Request ID aos logs
	logEntry := log.WithFields(logrus.Fields{
		"app_name":       "LambdaSMStoSNS",
		"aws_request_id": awsRequestID,
	})

	phoneNumber := event.QueryStringParameters["phone_number"]
	message := event.QueryStringParameters["message"]

	if phoneNumber == "" || message == "" {
		logEntry.Warn("phone_number or message is missing in query parameters")
		return Response{StatusCode: 400, Body: "phone_number or message is missing in query parameters"}, nil
	}

	sess, err := session.NewSession()
	if err != nil {
		logEntry.Errorf("Failed to create a new AWS session: %s", err.Error())
		return Response{StatusCode: 500, Body: "Failed to create a new AWS session"}, nil
	}

	snsClient := sns.New(sess)

	params := &sns.PublishInput{
		PhoneNumber: aws.String(phoneNumber),
		Message:     aws.String(message),
	}

	response, err := snsClient.Publish(params)
	if err != nil {
		logEntry.Errorf("Erro ao enviar SMS: %s", err.Error())
		return Response{StatusCode: 500, Body: "Erro ao enviar SMS"}, nil
	}

	logEntry.Infof("SMS enviado com sucesso para %s. MessageId: %s", phoneNumber, *response.MessageId)

	return Response{
		StatusCode: 200,
		Body:       fmt.Sprintf("SMS enviado com sucesso para %s. MessageId: %s", phoneNumber, *response.MessageId),
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
