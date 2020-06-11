# GripMock
GripMock is a **mock server** for **GRPC** services. It's using `.proto` file to generate implementation of gRPC service for you.
If you already familiar with [Apiary](https://apiary.io) or [WireMock](http://wiremock.org) for mocking API service and looking for similiar thing for GRPC then this is the perfect fit for that.


## How It Works
GripMock has 2 main components:
1. GRPC server that serving on `tcp://localhost:4770`. It's main job is to serve incoming rpc call from client then parse the input so that can be posted to Stub service to find the perfect stub match.
2. Stub server that serving on `http://localhost:4771`. It's main job is to store all the stub mapping. We can add a new stub or list existing stub using http request.

Matched stub will be returned to GRPC service then further parse it to response the rpc call.

## Quick Usage
 - Установить докер [Docker](https://docs.docker.com/install/)
 - Склонируйте этот репозиторий. В корне находится скрипт build.sh. Запустите его командой `sh build.sh latest`.
	Данная команда соберет вам контейнер, который в дальнейшем можно будет 	использовать. 
- В корне также находится ещё один скрипт, который уже запускает локальный мок сервер. Но для его коректной работы необходимо перенести прото файлы в отдельную папку и немного их изменить. После этого в скрипте нужно указать путь до вашей папки, вместо  `/Users/mobile-ci/Desktop/proto_server`
Check [`example`](https://github.com/tokopedia/gripmock/tree/master/example) folder for various usecase of gripmock.

##  Прото файлы.
- Создайте любую папку, в которой будут храниться ваши прото файлы. 
- Внутри создаем три папки: `Auth`, `MortgageBroker`, `SharedSubmodules`. 
- для iOS: идём в директорию проекта realty-ecosystem. Заходим в папки Auth и MortgageBroker. Из каждой из них копируем папку Submodules и переносим её внутрь соответсвующих папок. Что касается SharedSubmodules, то просто копируем содержимое и переносим. Должно получится следующее: Auth содержит папку Submodules, Submodules содержит auth-api, otp-api. Каждая из них содержит некоторое количество файлов и папок. Всё это можно удалить, кроме папки proto.
- для Android: Те же самые шаги, только берем нужные папки из репозиториев сервисов.
- Для того чтобы сервер запустился, необходимо изменение в некоторых протофайлах. Заходим в  `~/proto_server/MortgageBroker/Submodules/mortgage-api/proto/ru/vtblife/mortgage/v1/banks/model/banks_mortgage_offers.proto`. В нём заменяем message `MortgageOffer`  на `BankMortgageOffer` и заменяем тип переменных в этой файле на новый. Далее идём в `~/proto_server/MortgageBroker/Submodules/mortgage-api/proto/ru/vtblife/mortgage/v1/model/mortgage.proto` и удаляем строчку `import "ru/vtblife/mortgage/v1/banks/model/banks_mortgage_offers.proto";`, а так же строчку `vtblife.mortgage.v1.banks.model.Bank bank = 1;`. Всё, вы превосходны, можно запускать сервер и прогонять UI тесты. 

## Stubbing

Stubbing is the essential mocking of GripMock. It will match and return the expected result into GRPC service. This is where you put all your request expectation and response

### Dynamic stubbing
You could add stubbing on the fly with simple REST. HTTP stub server running on port `:4771`

- `GET /` Will list all stubs mapping.
- `POST /add` Will add stub with provided stub data
- `POST /find` Find matching stub with provided input. see [Input Matching](#input_matching) below.
- `GET /clear` Clear stub mappings.

Stub Format is JSON text format. It has skeleton like below:
```
{
  "service":"<servicename>", // name of service defined in proto
  "method":"<methodname>", // name of method that we want to mock
  "input":{ // input matching rule. see Input Matching Rule section below
    // put rule here
  },
  "output":{ // output json if input were matched
    "data":{
      // put result fields here
    },
    "error":"<error message>" // Optional. if you want to return error instead.
  }
}
```

For our `hello` service example we put stub with below text:
```
  {
    "service":"Greeter",
    "method":"SayHello",
    "input":{
      "equals":{
        "name":"gripmock"
      }
    },
    "output":{
      "data":{
        "message":"Hello GripMock"
      }
    }
  }
```

### Static stubbing
You could initialize gripmock with stub json files and provide the path using `--stub` argument. For example you may 
mount  your stub file in `/mystubs` folder then mount it to docker like
 
 `docker run -p 4770:4770 -p 4771:4771 -v /mypath:/proto -v /mystubs:/stub tkpd/gripmock --stub=/stub /proto/hello.proto`
 
Please note that Gripmock still serve http stubbing to modify stored stubs on the fly.
 
## <a name="input_matching"></a>Input Matching
Stub will responding the expected response if only requested with matching rule of input. Stub service will serve `/find` endpoint with format:
```
{
  "service":"<service name>",
  "method":"<method name>",
  "data":{
    // input that suppose to match with stored stubs
  }
}
```
So if you do `curl -X POST -d '{"service":"Greeter","method":"SayHello","data":{"name":"gripmock"}}' localhost:4771/find` stub service will find a match from listed stubs stored there.

### Input Matching Rule
Input matching has 3 rules to match an input. which is **equals**,**contains** and **regex**
<br>
**equals** will match the exact field name and value of input into expected stub. example stub JSON:
```
{
  .
  .
  "input":{
    "equals":{
      "name":"gripmock"
    }
  }
  .
  .
}
```

**contains** will match input that has the value declared expected fields. example stub JSON:
```
{
  .
  .
  "input":{
    "contains":{
      "field2":"hello"
    }
  }
  .
  .
}
```

**matches** using regex for matching fields expectation. example:

```
{
  .
  .
  "input":{
    "matches":{
      "name":"^grip.*$"
    }
  }
  .
  .
}
```

