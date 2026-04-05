// Package handler provides HandlerFunc
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"testing"

	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/risbern21/api_gateway/internal/database"
	"github.com/risbern21/api_gateway/internal/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	database.Setup()
	database.Client().Logger.LogMode(0)
	migrations.AutoMigrate()

	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestCreateUser(t *testing.T) {
	secretKey := "12345678901234567890123456789012"

	h := NewHandler(secretKey)
	handler := mux.NewRouter()
	handler.HandleFunc("/api/auth/signin", h.CreateUser)

	type args struct {
		req *http.Request
	}

	tests := []struct {
		name         string
		args         func(t *testing.T) args
		expectedCode int
		expectedBody []byte
	}{
		{
			name: "must return status 400 Bad Request on invalid body (missing email)",
			args: func(t *testing.T) args {
				reqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(1000)),
					Role:     "customer",
					Password: "DoeJohn",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(reqBody)
				require.NoError(t, err, "check body marshalling eror")

				req, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on request")

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("invalid request body\n"),
		},
		{
			name: "must return status 201 http.StatusCreated for valid user",
			args: func(t *testing.T) args {
				reqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(1000)),
					Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(1000)),
					Role:     "customer",
					Password: "DoeJohn",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(reqBody)
				require.NoError(t, err, "check body marshalling eror")

				req, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on request")

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusCreated,
			expectedBody: nil,
		},
		{
			name: "must return status 500 http.StatusInternalServerError for correct request body",
			args: func(t *testing.T) args {
				reqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(100_000)),
					Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(100_000)),
					Role:     "customer",
					Password: "DoeJohn",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(reqBody)
				require.NoError(t, err, "check body marshalling eror")

				firstReq, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on first request")
				firstReq.Header.Set("Content-Type", "application/json")
				firstRec := httptest.NewRecorder()
				handler.ServeHTTP(firstRec, firstReq)
				require.Equal(t, http.StatusCreated, firstRec.Code, "first signup should succeed")

				req, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on request")

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusInternalServerError,
			expectedBody: []byte("error creating user\n"),
		},
		// --- new cases ---
		{
			name: "must return status 400 Bad Request on invalid body (missing username)",
			args: func(t *testing.T) args {
				reqBody := &SigninRequest{
					Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(100_000)),
					Role:     "customer",
					Password: "DoeJohn",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(reqBody)
				require.NoError(t, err, "check body marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on request")

				return args{req: req}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("invalid request body\n"),
		},
		{
			name: "must return status 400 Bad Request on invalid body (missing password)",
			args: func(t *testing.T) args {
				reqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(100_000)),
					Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(100_000)),
					Role:     "customer",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(reqBody)
				require.NoError(t, err, "check body marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on request")

				return args{req: req}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("invalid request body\n"),
		},
		{
			name: "must return status 400 Bad Request on invalid body (missing role)",
			args: func(t *testing.T) args {
				reqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(100_000)),
					Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(100_000)),
					Password: "DoeJohn",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(reqBody)
				require.NoError(t, err, "check body marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on request")

				return args{req: req}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("invalid request body\n"),
		},
		{
			name: "must return status 400 Bad Request on malformed JSON body",
			args: func(t *testing.T) args {
				req, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBufferString("{not valid json"))
				require.NoError(t, err, "check for error on request")

				return args{req: req}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("error creating user\n"),
		},
		{
			name: "must return status 400 Bad Request on invalid email format",
			args: func(t *testing.T) args {
				reqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(100_000)),
					Email:    "not-a-valid-email",
					Role:     "customer",
					Password: "DoeJohn",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(reqBody)
				require.NoError(t, err, "check body marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on request")

				return args{req: req}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("invalid request body\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args(t)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, args.req)

			assert.Equal(t, tt.expectedCode, resp.Result().StatusCode, "check status code")

			if resp.Result().StatusCode == http.StatusCreated {
				signinResponse := &SigninResponse{}
				err := json.NewDecoder(resp.Body).Decode(&signinResponse)
				require.NoError(t, err, "check for response body decoding error")

				assert.NotEmpty(t, signinResponse.ID)
				assert.NotEmpty(t, signinResponse.Username)
				assert.NotEmpty(t, signinResponse.Email)
				assert.NotEmpty(t, signinResponse.Address)
				assert.NotEmpty(t, signinResponse.Phone)
				assert.NotEmpty(t, signinResponse.Role)
			} else {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "check for body parsing error")

				assert.Equal(t, tt.expectedBody, body, "check response body")
			}
		})
	}
}

func TestLogin(t *testing.T) {
	secretKey := "12345678901234567890123456789012"

	h := NewHandler(secretKey)
	handler := mux.NewRouter()
	handler.HandleFunc("/api/auth/login", h.Login)
	handler.HandleFunc("/api/auth/signin", h.CreateUser)

	type args struct {
		req *http.Request
	}

	tests := []struct {
		name         string
		args         func(t *testing.T) args
		expectedCode int
		expectedBody []byte
	}{
		{
			name: "Must return status 200 OK on correct request body",
			args: func(t *testing.T) args {
				//signup
				signupReqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(100_000)),
					Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(100_000)),
					Role:     "customer",
					Password: "DoeJohn",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(signupReqBody)
				require.NoError(t, err, "check body marshalling eror")

				signupReq, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on first request")
				signupReq.Header.Set("Content-Type", "application/json")
				signupRec := httptest.NewRecorder()
				handler.ServeHTTP(signupRec, signupReq)
				require.Equal(t, http.StatusCreated, signupRec.Code, "signup should succeed")

				loginReqBody := &LoginReq{
					Email:    signupReqBody.Email,
					Password: signupReqBody.Password,
				}

				reqBody, err := json.Marshal(loginReqBody)
				require.NoError(t, err, "Check json marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
				require.NoError(t, err, "Check for request error")

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusOK,
			expectedBody: nil,
		},
		{
			name: "Must return status 400 http.StatusBadRequest on invalid request body (missing email)",
			args: func(t *testing.T) args {
				body := &LoginReq{
					Password: "DoeJohn",
				}

				reqBody, err := json.Marshal(body)
				require.NoError(t, err, "Check json marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
				require.NoError(t, err, "Check for request error")

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("error invalid request body\n"),
		},
		{
			name: "Must return status 400 http.StatusBadRequest on invalid request body (missing password)",
			args: func(t *testing.T) args {
				body := &LoginReq{
					Email: "johnDoe856@gmail.com",
				}

				reqBody, err := json.Marshal(body)
				require.NoError(t, err, "Check json marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
				require.NoError(t, err, "Check for request error")

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("error invalid request body\n"),
		},
		{
			name: "Must return status 400 http.StatusBadRequest on invalid request body (missing password)",
			args: func(t *testing.T) args {
				body := &LoginReq{
					Email:    "user_not_found@gmail.com",
					Password: "12345",
				}

				reqBody, err := json.Marshal(body)
				require.NoError(t, err, "Check json marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
				require.NoError(t, err, "Check for request error")

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusNotFound,
			expectedBody: []byte("error user not found\n"),
		},
		// --- new cases ---
		{
			name: "Must return status 400 http.StatusBadRequest on wrong password",
			args: func(t *testing.T) args {
				// signup first
				signupReqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(100_000)),
					Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(100_000)),
					Role:     "customer",
					Password: "CorrectPassword",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(signupReqBody)
				require.NoError(t, err, "check body marshalling error")

				signupReq, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on signup request")
				signupReq.Header.Set("Content-Type", "application/json")
				signupRec := httptest.NewRecorder()
				handler.ServeHTTP(signupRec, signupReq)
				require.Equal(t, http.StatusCreated, signupRec.Code, "signup should succeed")

				// attempt login with wrong password
				loginReqBody := &LoginReq{
					Email:    signupReqBody.Email,
					Password: "WrongPassword",
				}

				reqBody, err := json.Marshal(loginReqBody)
				require.NoError(t, err, "check json marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
				require.NoError(t, err, "check for request error")

				return args{req: req}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("error invalid password\n"),
		},
		{
			name: "Must return status 400 http.StatusBadRequest on malformed JSON body",
			args: func(t *testing.T) args {
				req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBufferString("{bad json"))
				require.NoError(t, err, "check for request error")

				return args{req: req}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("error invalid request body\n"),
		},
		{
			name: "Must return status 400 http.StatusBadRequest on invalid email format",
			args: func(t *testing.T) args {
				body := &LoginReq{
					Email:    "not-an-email",
					Password: "somepassword",
				}

				reqBody, err := json.Marshal(body)
				require.NoError(t, err, "check json marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
				require.NoError(t, err, "check for request error")

				return args{req: req}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("error invalid request body\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args(t)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, args.req)

			assert.Equal(t, tt.expectedCode, resp.Result().StatusCode, "Check response status code")

			if resp.Result().StatusCode == http.StatusOK {
				loginRes := &LoginRes{}
				err := json.NewDecoder(resp.Body).Decode(&loginRes)
				require.NoError(t, err, "Check json decoding error")

				assert.NotEmpty(t, loginRes.SessionID)
				assert.NotEmpty(t, loginRes.AccessToken)
				assert.NotEmpty(t, loginRes.AccessTokenExpiresAt)
				assert.NotEmpty(t, loginRes.RefreshToken)
				assert.NotEmpty(t, loginRes.RefreshTokenExpiresAt)
				assert.NotEmpty(t, loginRes.User)
			} else {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "check body parsing error")
				assert.Equal(t, tt.expectedBody, body, "check response body")
			}
		})
	}
}

func TestLogout(t *testing.T) {
	secretKey := "12345678901234567890123456789012"

	h := NewHandler(secretKey)
	handler := mux.NewRouter()
	handler.HandleFunc("/api/auth/logout", h.Logout)
	handler.HandleFunc("/api/auth/signin", h.CreateUser)
	handler.HandleFunc("/api/auth/login", h.Login)

	type args struct {
		req *http.Request
	}

	tests := []struct {
		name         string
		args         func(t *testing.T) args
		expectedCode int
		expectedBody []byte
	}{
		{
			name: "Return status 200 http.StatusOK on passing a valid request",
			args: func(t *testing.T) args {
				//signup
				signupReqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(100_000)),
					Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(100_000)),
					Role:     "customer",
					Password: "DoeJohn",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(signupReqBody)
				require.NoError(t, err, "check body marshalling eror")

				signupReq, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on first request")
				signupReq.Header.Set("Content-Type", "application/json")
				signupRec := httptest.NewRecorder()
				handler.ServeHTTP(signupRec, signupReq)
				require.Equal(t, http.StatusCreated, signupRec.Code, "signup should succeed")

				signupRes := &SigninResponse{}
				err = json.NewDecoder(signupRec.Body).Decode(&signupRes)
				require.NoError(t, err, "check for decoding error")

				//login
				loginReqBody := &LoginReq{
					Email:    signupReqBody.Email,
					Password: signupReqBody.Password,
				}

				body, err = json.Marshal(loginReqBody)
				require.NoError(t, err, "check for login request body marshalling error")

				loginReq, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
				require.NoError(t, err, "check for login reuest error")
				loginReq.Header.Set("Content-Type", "application/json")
				loginRec := httptest.NewRecorder()
				handler.ServeHTTP(loginRec, loginReq)
				require.Equal(t, http.StatusOK, loginRec.Code, "login should succeed")

				loginRes := &LoginRes{}
				err = json.NewDecoder(loginRec.Body).Decode(&loginRes)
				require.NoError(t, err, "check for login response decoding error")

				//logout
				req, err := http.NewRequest("POST", "/api/auth/logout", nil)
				require.NoError(t, err, "check for request error")

				q := req.URL.Query()
				q.Add("id", loginRes.SessionID)
				req.URL.RawQuery = q.Encode()

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusNoContent,
			expectedBody: nil,
		},
		{
			name: "return status 400 http.StatusBadRequest on not passing session id",
			args: func(t *testing.T) args {
				req, err := http.NewRequest("POST", "/api/auth/logout", nil)
				require.NoError(t, err, "check for request error")

				q := req.URL.Query()
				q.Add("id", "")
				req.URL.RawQuery = q.Encode()

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("error no session id\n"),
		},
		{
			name: "return status 404 http.StatusNotFound on passing invalid session id",
			args: func(t *testing.T) args {
				req, err := http.NewRequest("POST", "/api/auth/logout", nil)
				require.NoError(t, err, "check for request error")

				q := req.URL.Query()
				q.Add("id", "nahman")
				req.URL.RawQuery = q.Encode()

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusNotFound,
			expectedBody: []byte("error session not found\n"),
		},
		// --- new cases ---
		{
			name: "return status 404 http.StatusNotFound when logging out an already deleted session",
			args: func(t *testing.T) args {
				// signup
				signupReqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(100_000)),
					Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(100_000)),
					Role:     "customer",
					Password: "DoeJohn",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(signupReqBody)
				require.NoError(t, err, "check body marshalling error")

				signupReq, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for signup request error")
				signupReq.Header.Set("Content-Type", "application/json")
				signupRec := httptest.NewRecorder()
				handler.ServeHTTP(signupRec, signupReq)
				require.Equal(t, http.StatusCreated, signupRec.Code, "signup should succeed")

				// login
				loginReqBody := &LoginReq{
					Email:    signupReqBody.Email,
					Password: signupReqBody.Password,
				}
				body, err = json.Marshal(loginReqBody)
				require.NoError(t, err, "check login body marshalling error")

				loginReq, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
				require.NoError(t, err, "check for login request error")
				loginReq.Header.Set("Content-Type", "application/json")
				loginRec := httptest.NewRecorder()
				handler.ServeHTTP(loginRec, loginReq)
				require.Equal(t, http.StatusOK, loginRec.Code, "login should succeed")

				loginRes := &LoginRes{}
				err = json.NewDecoder(loginRec.Body).Decode(&loginRes)
				require.NoError(t, err, "check login response decoding error")

				// first logout — deletes the session
				firstLogoutReq, err := http.NewRequest("POST", "/api/auth/logout", nil)
				require.NoError(t, err, "check for first logout request error")
				q := firstLogoutReq.URL.Query()
				q.Add("id", loginRes.SessionID)
				firstLogoutReq.URL.RawQuery = q.Encode()
				firstLogoutRec := httptest.NewRecorder()
				handler.ServeHTTP(firstLogoutRec, firstLogoutReq)
				require.Equal(t, http.StatusNoContent, firstLogoutRec.Code, "first logout should succeed")

				// second logout — session is already gone
				req, err := http.NewRequest("POST", "/api/auth/logout", nil)
				require.NoError(t, err, "check for second logout request error")
				q2 := req.URL.Query()
				q2.Add("id", loginRes.SessionID)
				req.URL.RawQuery = q2.Encode()

				return args{req: req}
			},
			expectedCode: http.StatusNotFound,
			expectedBody: []byte("error session not found\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args(t)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, args.req)

			assert.Equal(t, tt.expectedCode, resp.Result().StatusCode, "Check response status code")

			if resp.Result().StatusCode != http.StatusNoContent {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "check for body parsing error")

				assert.Equal(t, tt.expectedBody, body, "check response body")
			}
		})
	}
}

// signupAndLogin is a test helper that creates a user and logs them in,
// returning the full login response. It fails the test immediately on any error.
func signupAndLogin(t *testing.T, handler http.Handler) *LoginRes {
	t.Helper()

	signupReqBody := &SigninRequest{
		Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(100_000)),
		Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(100_000)),
		Role:     "customer",
		Password: "DoeJohn",
		Address:  "JD avenue,LA",
		Phone:    "12346912345",
	}

	body, err := json.Marshal(signupReqBody)
	require.NoError(t, err, "check signup body marshalling error")

	signupReq, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
	require.NoError(t, err, "check for signup request error")
	signupReq.Header.Set("Content-Type", "application/json")
	signupRec := httptest.NewRecorder()
	handler.ServeHTTP(signupRec, signupReq)
	require.Equal(t, http.StatusCreated, signupRec.Code, "signup should succeed")

	loginReqBody := &LoginReq{
		Email:    signupReqBody.Email,
		Password: signupReqBody.Password,
	}
	body, err = json.Marshal(loginReqBody)
	require.NoError(t, err, "check login body marshalling error")

	loginReq, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	require.NoError(t, err, "check for login request error")
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code, "login should succeed")

	loginRes := &LoginRes{}
	err = json.NewDecoder(loginRec.Body).Decode(&loginRes)
	require.NoError(t, err, "check login response decoding error")

	return loginRes
}

func TestRenewAccessToken(t *testing.T) {
	secretKey := "12345678901234567890123456789012"

	h := NewHandler(secretKey)
	handler := mux.NewRouter()
	handler.HandleFunc("/api/auth/renew", h.RenewAccessToken)
	handler.HandleFunc("/api/auth/logout", h.Logout)
	handler.HandleFunc("/api/tokens/revoke", h.RevokeSession)
	handler.HandleFunc("/api/auth/signin", h.CreateUser)
	handler.HandleFunc("/api/auth/login", h.Login)

	type args struct {
		req *http.Request
	}

	tests := []struct {
		name         string
		args         func(t *testing.T) args
		expectedCode int
		expectedBody []byte
	}{
		{
			name: "return status 200 http.StatusOK on valid refresh token",
			args: func(t *testing.T) args {
				loginRes := signupAndLogin(t, handler)

				renewReqBody := &RenewAccessTokenReq{
					RefreshToken: loginRes.RefreshToken,
				}
				body, err := json.Marshal(renewReqBody)
				require.NoError(t, err, "check renew body marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/renew", bytes.NewBuffer(body))
				require.NoError(t, err, "check for renew request error")
				req.Header.Set("Content-Type", "application/json")

				return args{req: req}
			},
			expectedCode: http.StatusOK,
			expectedBody: nil,
		},
		{
			name: "return status 400 http.StatusBadRequest on missing refresh token field",
			args: func(t *testing.T) args {
				// send an empty object — refresh_token field is absent
				req, err := http.NewRequest("POST", "/api/auth/renew", bytes.NewBufferString(`{}`))
				require.NoError(t, err, "check for request error")
				req.Header.Set("Content-Type", "application/json")

				return args{req: req}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("error invalid request body\n"),
		},
		{
			name: "return status 401 http.StatusUnauthorized for invalid refresh token",
			args: func(t *testing.T) args {
				renewReqBody := &RenewAccessTokenReq{
					RefreshToken: "this.is.not.a.valid.jwt",
				}
				body, err := json.Marshal(renewReqBody)
				require.NoError(t, err, "check body marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/renew", bytes.NewBuffer(body))
				require.NoError(t, err, "check for request error")
				req.Header.Set("Content-Type", "application/json")

				return args{req: req}
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: []byte("error unable to verifying token\n"),
		},
		// --- new cases ---
		{
			name: "return status 400 http.StatusBadRequest on malformed JSON body",
			args: func(t *testing.T) args {
				req, err := http.NewRequest("POST", "/api/auth/renew", bytes.NewBufferString("{bad json"))
				require.NoError(t, err, "check for request error")
				req.Header.Set("Content-Type", "application/json")

				return args{req: req}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("error invalid request body\n"),
		},
		{
			name: "return status 401 http.StatusUnauthorized on revoked session",
			args: func(t *testing.T) args {
				loginRes := signupAndLogin(t, handler)

				// revoke the session
				revokeReq, err := http.NewRequest("POST", "/api/tokens/revoke", nil)
				require.NoError(t, err, "check for revoke request error")
				q := revokeReq.URL.Query()
				q.Add("id", loginRes.SessionID)
				revokeReq.URL.RawQuery = q.Encode()
				revokeRec := httptest.NewRecorder()
				handler.ServeHTTP(revokeRec, revokeReq)
				require.Equal(t, http.StatusNoContent, revokeRec.Code, "session revoke should succeed")

				// try to renew with the now-revoked session's refresh token
				renewReqBody := &RenewAccessTokenReq{
					RefreshToken: loginRes.RefreshToken,
				}
				body, err := json.Marshal(renewReqBody)
				require.NoError(t, err, "check body marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/renew", bytes.NewBuffer(body))
				require.NoError(t, err, "check for request error")
				req.Header.Set("Content-Type", "application/json")

				return args{req: req}
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: []byte("session is revoked\n"),
		},
		{
			name: "return status 200 http.StatusOK and response contains non-empty access token and expiry",
			args: func(t *testing.T) args {
				loginRes := signupAndLogin(t, handler)

				renewReqBody := &RenewAccessTokenReq{
					RefreshToken: loginRes.RefreshToken,
				}
				body, err := json.Marshal(renewReqBody)
				require.NoError(t, err, "check body marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/renew", bytes.NewBuffer(body))
				require.NoError(t, err, "check for request error")
				req.Header.Set("Content-Type", "application/json")

				return args{req: req}
			},
			expectedCode: http.StatusOK,
			expectedBody: nil,
		},
		{
			name: "return status 404 http.StatusNotFound after session is deleted via logout",
			// NOTE: this test will currently fail due to a typo in handler.go:
			// http.StatusNotFOund should be http.StatusNotFound. Fix that first.
			args: func(t *testing.T) args {
				loginRes := signupAndLogin(t, handler)

				// logout — deletes the session entirely
				logoutReq, err := http.NewRequest("POST", "/api/auth/logout", nil)
				require.NoError(t, err, "check for logout request error")
				q := logoutReq.URL.Query()
				q.Add("id", loginRes.SessionID)
				logoutReq.URL.RawQuery = q.Encode()
				logoutRec := httptest.NewRecorder()
				handler.ServeHTTP(logoutRec, logoutReq)
				require.Equal(t, http.StatusNoContent, logoutRec.Code, "logout should succeed")

				// attempt renew — session no longer exists
				renewReqBody := &RenewAccessTokenReq{
					RefreshToken: loginRes.RefreshToken,
				}
				body, err := json.Marshal(renewReqBody)
				require.NoError(t, err, "check body marshalling error")

				req, err := http.NewRequest("POST", "/api/auth/renew", bytes.NewBuffer(body))
				require.NoError(t, err, "check for request error")
				req.Header.Set("Content-Type", "application/json")

				return args{req: req}
			},
			expectedCode: http.StatusNotFound,
			expectedBody: []byte("error session not found\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args(t)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, args.req)

			assert.Equal(t, tt.expectedCode, resp.Result().StatusCode, "Check response status code")

			if resp.Result().StatusCode == http.StatusOK {
				renewRes := &RenewAccessTokenRes{}
				err := json.NewDecoder(resp.Body).Decode(&renewRes)
				require.NoError(t, err, "check renew response decoding error")

				assert.NotEmpty(t, renewRes.AccessToken, "access token must not be empty")
				assert.NotEmpty(t, renewRes.AccessTokenExpiresAt, "access token expiry must not be empty")
			} else {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "check for body parsing error")

				assert.Equal(t, tt.expectedBody, body, "check response body")
			}
		})
	}
}

func TestRevokeSession(t *testing.T) {
	secretKey := "12345678901234567890123456789012"

	h := NewHandler(secretKey)
	handler := mux.NewRouter()
	handler.HandleFunc("/api/tokens/revoke", h.RevokeSession)
	handler.HandleFunc("/api/auth/signin", h.CreateUser)
	handler.HandleFunc("/api/auth/login", h.Login)

	type args struct {
		req *http.Request
	}

	tests := []struct {
		name         string
		args         func(t *testing.T) args
		expectedCode int
		expectedBody []byte
	}{
		{
			name: "return status 400 http.StatusBadRequest on no session id",
			args: func(t *testing.T) args {
				req, err := http.NewRequest("POST", "/api/tokens/revoke", nil)
				require.NoError(t, err, "check for reqeust error")

				q := req.URL.Query()
				q.Add("id", "")
				req.URL.RawQuery = q.Encode()

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: []byte("error no session id\n"),
		},
		{
			name: "return status 204 http.StatusNoContent on valid request",
			args: func(t *testing.T) args {
				//signup
				signupReqBody := &SigninRequest{
					Username: fmt.Sprintf("JohnDoe69%v", rand.Intn(100_000)),
					Email:    fmt.Sprintf("johnDoe%v@gmail.com", rand.Intn(100_000)),
					Role:     "customer",
					Password: "DoeJohn",
					Address:  "JD avenue,LA",
					Phone:    "12346912345",
				}

				body, err := json.Marshal(signupReqBody)
				require.NoError(t, err, "check body marshalling eror")

				signupReq, err := http.NewRequest("POST", "/api/auth/signin", bytes.NewBuffer(body))
				require.NoError(t, err, "check for error on first request")
				signupReq.Header.Set("Content-Type", "application/json")
				signupRec := httptest.NewRecorder()
				handler.ServeHTTP(signupRec, signupReq)
				require.Equal(t, http.StatusCreated, signupRec.Code, "signup should succeed")

				signupRes := &SigninResponse{}
				err = json.NewDecoder(signupRec.Body).Decode(&signupRes)
				require.NoError(t, err, "check for decoding error")

				//login
				loginReqBody := &LoginReq{
					Email:    signupReqBody.Email,
					Password: signupReqBody.Password,
				}

				body, err = json.Marshal(loginReqBody)
				require.NoError(t, err, "check for login request body marshalling error")

				loginReq, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
				require.NoError(t, err, "check for login reuest error")
				loginReq.Header.Set("Content-Type", "application/json")
				loginRec := httptest.NewRecorder()
				handler.ServeHTTP(loginRec, loginReq)
				require.Equal(t, http.StatusOK, loginRec.Code, "login should succeed")

				loginRes := &LoginRes{}
				err = json.NewDecoder(loginRec.Body).Decode(&loginRes)
				require.NoError(t, err, "check for login response decoding error")

				// revoke
				revokeReq, err := http.NewRequest("POST", "/api/tokens/revoke", nil)
				require.NoError(t, err, "check for revoke request error")

				q := revokeReq.URL.Query()
				q.Add("id", loginRes.SessionID)
				revokeReq.URL.RawQuery = q.Encode()

				return args{
					req: revokeReq,
				}
			},
			expectedCode: http.StatusNoContent,
			expectedBody: nil,
		},
		{
			name: "return status 404 http.StatusNotFound on invalid session id",
			args: func(t *testing.T) args {
				req, err := http.NewRequest("POST", "/api/tokens/revoke", nil)
				require.NoError(t, err, "check for revoke request error")

				q := req.URL.Query()
				q.Add("id", "crazy-ass-punk-ass-niyega")
				req.URL.RawQuery = q.Encode()

				return args{
					req: req,
				}
			},
			expectedCode: http.StatusNotFound,
			expectedBody: []byte("error unable to find session\n"),
		},
		// --- new cases ---
		{
			name: "return status 204 http.StatusNoContent revoking an already-revoked session (idempotent)",
			args: func(t *testing.T) args {
				loginRes := signupAndLogin(t, handler)

				// first revoke
				firstRevokeReq, err := http.NewRequest("POST", "/api/tokens/revoke", nil)
				require.NoError(t, err, "check for first revoke request error")
				q := firstRevokeReq.URL.Query()
				q.Add("id", loginRes.SessionID)
				firstRevokeReq.URL.RawQuery = q.Encode()
				firstRevokeRec := httptest.NewRecorder()
				handler.ServeHTTP(firstRevokeRec, firstRevokeReq)
				require.Equal(t, http.StatusNoContent, firstRevokeRec.Code, "first revoke should succeed")

				// second revoke on the same session
				req, err := http.NewRequest("POST", "/api/tokens/revoke", nil)
				require.NoError(t, err, "check for second revoke request error")
				q2 := req.URL.Query()
				q2.Add("id", loginRes.SessionID)
				req.URL.RawQuery = q2.Encode()

				return args{req: req}
			},
			// RevokeSession fetches then sets IsRevoked=true. The session still
			// exists in the DB after the first revoke, so a second call succeeds.
			expectedCode: http.StatusNoContent,
			expectedBody: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args(t)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, args.req)

			assert.Equal(t, tt.expectedCode, resp.Result().StatusCode, "Check response status code")

			if resp.Result().StatusCode != http.StatusNoContent {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "check for body parsing error")

				assert.Equal(t, tt.expectedBody, body, "check response body")
			}
		})
	}
}
