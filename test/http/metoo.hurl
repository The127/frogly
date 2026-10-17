# an idea to say me too to
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl garden", "visibility": "private"}
HTTP 201
[Captures]
project: jsonpath "$.id"

POST {{frogly}}/api/projects/{{project}}/ideas
Authorization: Bearer {{token}}
Content-Type: application/json
{"text": "search by any word"}
HTTP 201
[Captures]
idea: jsonpath "$.id"

# the planter's me too answers with the idea, and they still count once
POST {{frogly}}/api/projects/{{project}}/ideas/{{idea}}/metoo
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
header "Content-Type" == "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.id" == "{{idea}}"
jsonpath "$.people" == 1

# an idea that doesn't exist is not found
POST {{frogly}}/api/projects/{{project}}/ideas/01a14180-d957-7cc5-af29-7f1fe18d89bb/metoo
Authorization: Bearer {{token}}
HTTP 404
[Asserts]
header "Content-Type" == "application/problem+json"

# without a token, frogly asks for one
POST {{frogly}}/api/projects/{{project}}/ideas/{{idea}}/metoo
HTTP 401
[Asserts]
header "WWW-Authenticate" == "Bearer"
