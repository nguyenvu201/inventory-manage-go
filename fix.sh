#!/bin/bash
FILE="internal/service/impl/report_service_test.go"
sed -i '' 's/"inventory-manage\/internal\/service\/impl"/"inventory-manage\/internal\/service\/impl"\n\t"inventory-manage\/pkg\/logger"/g' "$FILE"
sed -i '' 's/global.Logger = zap.NewNop()/global.Logger = \&logger.LoggerZap{Logger: zap.NewNop()}/g' "$FILE"
sed -i '' 's/mr.List()/mr.Keys("*")/g' "$FILE"
