# a private project to change
POST {{frogly}}/api/projects
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "Hurl garden", "visibility": "private"}
HTTP 201
[Captures]
id: jsonpath "$.id"

# a member makes it public and gets the project back
PUT {{frogly}}/api/projects/{{id}}/visibility
Authorization: Bearer {{token}}
Content-Type: application/json
{"visibility": "public"}
HTTP 200
[Asserts]
header "Content-Type" == "application/json"
jsonpath "$.id" == "{{id}}"
jsonpath "$.visibility" == "public"

# a visibility frogly doesn't know says what's wrong
PUT {{frogly}}/api/projects/{{id}}/visibility
Authorization: Bearer {{token}}
Content-Type: application/json
{"visibility": "secret"}
HTTP 400
[Asserts]
header "Content-Type" == "application/problem+json"
jsonpath "$.detail" == "a project is public or private"

# a project that doesn't exist is not found
PUT {{frogly}}/api/projects/01a14180-d957-7cc5-af29-7f1fe18d89bb/visibility
Authorization: Bearer {{token}}
Content-Type: application/json
{"visibility": "public"}
HTTP 404
