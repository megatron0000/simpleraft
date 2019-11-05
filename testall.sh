# Tests in order from low-level to high-level
go test ./storage -failfast -count 2
go test ./status -failfast -count 2