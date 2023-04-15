# lambdaGo-AWS-SMSToSNS
Lambda GO, Implementação de envio de SMS a partir do SNS.

Use:

Input lambda:

Type Json

{
"queryStringParameters": {
"phone_number": "+5511999999999",
"message": "Teste de envio de SMS via AWS LambdaGOLANG e SNS!"
}
}
