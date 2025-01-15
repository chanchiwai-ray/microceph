PUT http://localhost/1.0/ops/maintenance
{
	"Status": "unknown"
}
HTTP 200
[Asserts]
jsonpath "$.error_code" == 500
jsonpath "$.error" == "unknown status encounter: 'unknown', can only be 'maintenance' or 'non-maintenance'"
