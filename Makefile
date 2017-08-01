testkeys:
	mkdir testdata
	openssl genrsa -out testdata/private.pem 4096
	openssl rsa -in testdata/private.pem -pubout -outform PEM -out testdata/public.pem
