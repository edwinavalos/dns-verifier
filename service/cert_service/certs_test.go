package cert_service

//func Test_requestCertificate(t *testing.T) {
//	testConfig := config.config{}
//	testConfig.LESettings.PrivateKeyLocation = "C:\\mastodon\\private-key.pem"
//	testConfig.LESettings.CADirURL = lego.LEDirectoryStaging
//	testConfig.LESettings.KeyAuth = "asufficientlylongenoughstringwithenoughentropy"
//	testConfig.DB = config.DatabaseSettings{
//		TableName: "dns-verifier-test",
//		Region:    "us-east-1",
//		IsLocal:   true,
//	}
//	cfg = &testConfig
//	log := logger.Logger{Logger: zerolog.Logger{}}
//	storage.SetConfig(&testConfig)
//	datastore2.SetLogger(&log)
//	SetLogger(&log)
//
//	dbStorage, err := dynamo.NewStorage(&testConfig)
//	if err != nil {
//		t.Fatal(err)
//	}
//	err = dbStorage.Initialize()
//	if err != nil {
//		t.Fatal(err)
//	}
//	dbStorage = dbStorage
//	type args struct {
//		domain string
//		email  string
//		userId string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "secondtest.amoslabs.cloud txt record",
//			args: args{
//				domain: "secondtest.amoslabs.cloud",
//				email:  "admin@amoslabs.cloud",
//				userId: uuid.New().String(),
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			fqdn, key, err := RequestCertificate(tt.args.userId, tt.args.domain, tt.args.email)
//			if err != nil {
//				t.Fatal(err)
//			}
//			t.Logf("%s %s", fqdn, key)
//		})
//	}
//}

//func Test_completeCertificateRequest(t *testing.T) {
//	testConfig := config.config{}
//	testConfig.LESettings = config.LetsEncryptSettings{
//		AdminEmail:         "admin@amoslabs.cloud",
//		PrivateKeyLocation: "C:\\mastodon\\private-key.pem",
//		KeyAuth:            "asufficientlylongenoughstringwithenoughentropy",
//		CADirURL:           lego.LEDirectoryStaging,
//	}
//	testConfig.DB = config.DatabaseSettings{
//		TableName: "dns-verifier-test",
//		Region:    "us-east-1",
//		IsLocal:   true,
//	}
//	cfg = &testConfig
//	storage.SetConfig(&testConfig)
//	log := logger.Logger{Logger: zerolog.Logger{}}
//	SetLogger(&log)
//	datastore2.SetLogger(&log)
//
//	dbStorage, err := dynamo.NewStorage(&testConfig)
//	if err != nil {
//		t.Fatal(err)
//	}
//	err = dbStorage.Initialize()
//	if err != nil {
//		t.Fatal(err)
//	}
//	dbStorage = dbStorage
//
//	type args struct {
//		domain string
//		email  string
//		userId string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    *certificate.Resource
//		wantErr bool
//	}{
//		{
//			name: "secondtest.amoslabs.cloud",
//			args: args{
//				domain: "secondtest.amoslabs.cloud",
//				email:  "admin@amoslabs.cloud",
//				userId: "2c84b63c-9a96-11ed-a8fc-0242ac120002",
//			},
//			want:    nil,
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			domainInformation := models.DomainInformation{
//				DomainName: tt.args.domain,
//				UserID:     tt.args.userId,
//			}
//			err := dbStorage.PutDomainInfo(domainInformation)
//			if err != nil {
//				t.Error(err)
//			}
//			zone, key, err := RequestCertificate(tt.args.userId, tt.args.domain, tt.args.email)
//			if err != nil {
//				t.Errorf("RequestCertificate() err %v, wantErr: %v", err, tt.wantErr)
//				return
//			}
//			t.Logf("zone: %s key: %s", zone, key)
//
//			time.Sleep(300 * time.Second)
//
//			err := CompleteCertificateRequest(tt.args.userId, tt.args.domain, tt.args.email)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("CompleteCertificateRequest() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			for _, der := range ders {
//				cert, err := x509.ParseCertificate(der)
//				if err != nil {
//					return
//				}
//				t.Logf("Certificate: \n%s", string(cert.Raw))
//			}
//		})
//	}
//}
