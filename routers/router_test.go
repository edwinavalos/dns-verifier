package routers

//// TestAllOfIt this is an ugly happy path test through registration to the website
//func TestAllOfIt(t *testing.T) {
//
//	// Configure the application
//	domainName := "test.edwinavalos.com"
//
//	viper.SetConfigName("config-test")
//	viper.SetConfigType("yaml")
//	viper.AddConfigPath("./../resources")
//	err := viper.ReadInConfig()
//	if err != nil {
//		t.Fatal("unable to read configuration file, exiting.")
//	}
//	ctx := context.Background()
//
//	appConfig := config.NewConfig()
//	appConfig.RootCtx = ctx
//	appConfig.ReadConfig()
//
//	// Create our router
//	r := InitRouter()
//
//	// First we create a new domain in our service
//	w := httptest.NewRecorder()
//	newDomainRequest := v1.CreateDomainInformationReq{
//		DomainName: domainName,
//		UserId:     uuid.New(),
//	}
//	var buf bytes.Buffer
//	err = json.NewEncoder(&buf).Encode(newDomainRequest)
//	if err != nil {
//		t.Fatal(err)
//	}
//	req, err := http.NewRequest("POST", "/api/v1/domain", &buf)
//	if err != nil {
//		t.Fatal(err)
//	}
//	r.ServeHTTP(w, req)
//
//	assert.Equal(t, w.Code, http.StatusOK)
//
//	// Then we generate an ownership key for verification
//	buf = bytes.Buffer{}
//	w = httptest.NewRecorder()
//	newGenerateOwnershipKey := v1.GenerateOwnershipKeyReq{DomainName: domainName}
//	err = json.NewEncoder(&buf).Encode(newGenerateOwnershipKey)
//	req, err = http.NewRequest("POST", "/api/v1/domain/verificationKey", &buf)
//	if err != nil {
//		t.Fatal(err)
//	}
//	r.ServeHTTP(w, req)
//
//	assert.Equal(t, w.Code, http.StatusOK)
//
//	// Now we need to change the domain information we just wrote to be one that we can verify with our
//	// edwinavalos.com domain
//	di := models.DomainInformation{DomainName: domainName}
//	diToUpdate, err := di.Load(ctx)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	diToUpdate.Verification.VerificationKey = "111122223333"
//	diToUpdate.Delegations.ARecords = []string{"34.217.225.52"}
//	diToUpdate.Delegations.CNames = []string{"spons.us"}
//	// Save it so the services will have access to it
//	err = diToUpdate.SaveDomainInformation(ctx)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	// Verify the domains various records
//	buf = bytes.Buffer{}
//	w = httptest.NewRecorder()
//
//	// Verify that we own the domain
//
//	req, err = http.NewRequest("POST", fmt.Sprintf("/api/v1/domain/verification?domain_name=%s", domainName), &buf)
//	if err != nil {
//		t.Fatal(err)
//	}
//	r.ServeHTTP(w, req)
//
//	assert.Equal(t, w.Code, http.StatusOK)
//
//	buf = bytes.Buffer{}
//	w = httptest.NewRecorder()
//
//	// Verify that our ARecord points to a service node
//	delegationReq1 := v1.VerifyDelegationReq{
//		DomainName: domainName,
//		Type:       v1.ARecord,
//	}
//
//	err = json.NewEncoder(&buf).Encode(delegationReq1)
//
//	req, err = http.NewRequest("POST", "/api/v1/domain/verification", &buf)
//	if err != nil {
//		t.Fatal(err)
//	}
//	r.ServeHTTP(w, req)
//
//	assert.Equal(t, w.Code, http.StatusOK)
//	// need to write one that does CNames with another name loaded in to the di object
//
//}
