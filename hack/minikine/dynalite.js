var AWS = require('aws-sdk')
var dynalite = require('dynalite')

var dynaliteServer = dynalite()

var envs = function (key, defaultValue) {
  if ('key' in process.env) {
    return process.env[key]
  } else {
    return defaultValue
  }
}

const SETTINGS = {
  'port': envs('MINIKINE_PORT', 4567),
  'region': envs('MINIKINE_REGION', 'eu-west-2'),
}

// Set up credentials in AWS-SDK
process.env.AWS_ACCESS_KEY_ID = 'XXXXXXXXXXXXXXXXXXX';
process.env.AWS_SECRET_ACCESS_KEY = 'XXXXXXXXXXXXXXXXXXXXXXXXXX';
process.env.AWS_REGION = SETTINGS.region;

// Start server
dynaliteServer.listen(SETTINGS.port, function (err) {
  if (err) {
    throw err
  }
  console.log('Dynalite started on port ' + SETTINGS.port)

  var dynamo = new AWS.DynamoDB({ endpoint: 'http://127.0.0.1:' + SETTINGS.port, region: SETTINGS.region })
  bootstrap(dynamo)

  console.log('Bootstrap finished!')
})

function bootstrap(dynamo) {
  tables = {
    'rdss_am_checkpoints': { 'key': 'Shard' },
    'rdss_am_clients': { 'key': 'ID' },
    'rdss_am_metadata': { 'key': 'Key' },
  }
  for (var prop in tables) {
    key = tables[prop].key

    params = {
      TableName: prop,
      KeySchema: [
        { AttributeName: key, KeyType: "HASH" },
      ],
      AttributeDefinitions: [
        { AttributeName: key, AttributeType: "S" }
      ],
      ProvisionedThroughput: {
        ReadCapacityUnits: 10,
        WriteCapacityUnits: 10
      }
    }

    dynamo.createTable(params, function (err, data) {
      if (err) {
        console.error("Unable to create table. Error JSON:", JSON.stringify(err, null, 2));
      }
    })
  }
}