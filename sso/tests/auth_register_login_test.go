package tests

import (
	"sso/tests/suite"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/golang-jwt/jwt/v5"
	ssov1 "github.com/mergen1212/grpc_gen/gen/go/sso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	emptyAppID = 0
    appID = 1                 // ID приложения, которое мы создали миграцией
    appSecret = "test-secret" // Секретный ключ приложения
	passDefaultLen = 10
)

func randomFakePassword() string {
    return gofakeit.Password(true, true, true, true, false, passDefaultLen)
}

func TestRegisterLogin_Login_HappyPath(t *testing.T) {
    ctx, st := suite.New(t)

    email := gofakeit.Email()
    pass := randomFakePassword()

    // Сначала зарегистрируем нового пользователя, которого будем логинить
    respReg, err := st.AuthClient.Register(ctx, &ssov1.RegisterRequest{
        Email:    email,
        Password: pass,
    })
    // Это вспомогательный запрос, поэтому делаем лишь минимальные проверки
    require.NoError(t, err)
    assert.NotEmpty(t, respReg.GetUserId())

    // А это основная проверка
    respLogin, err := st.AuthClient.Login(ctx, &ssov1.LoginRequest{
        Email:    email,
        Password: pass,
        AppId:    appID,
    })
    require.NoError(t, err)

    token := respLogin.GetToken()
    require.NotEmpty(t, token) // Проверяем, что он не пустой

    // Отмечаем время, в которое бы выполнен логин.
    // Это понадобится для проверки TTL токена
    loginTime := time.Now()

    // Парсим и валидируем токен
    tokenParsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
        return []byte(appSecret), nil
    })
    // Если ключ окажется невалидным, мы получим соответствующую ошибку
    require.NoError(t, err)

    // Преобразуем к типу jwt.MapClaims, в котором мы сохраняли данные
    claims, ok := tokenParsed.Claims.(jwt.MapClaims)
    require.True(t, ok)

    // Проверяем содержимое токена
    assert.Equal(t, respReg.GetUserId(), int64(claims["uid"].(float64)))
    assert.Equal(t, email, claims["email"].(string))
    assert.Equal(t, appID, int(claims["app_id"].(float64)))

    const deltaSeconds = 1

    // Проверяем, что TTL токена примерно соответствует нашим ожиданиям.
    assert.InDelta(t, loginTime.Add(st.Cfg.TokenTTL).Unix(), claims["exp"].(float64), deltaSeconds)
}

func TestRegisterLogin_DuplicatedRegistration(t *testing.T) {
    ctx, st := suite.New(t)

    email := gofakeit.Email()
    pass := randomFakePassword()

    // Первая попытка должна быть успешной
    respReg, err := st.AuthClient.Register(ctx, &ssov1.RegisterRequest{
        Email:    email,
        Password: pass,
    })
    require.NoError(t, err)
    require.NotEmpty(t, respReg.GetUserId())

    // Вторая попытка - фэил
    respReg, err = st.AuthClient.Register(ctx, &ssov1.RegisterRequest{
        Email:    email,
        Password: pass,
    })
    require.Error(t, err)
    assert.Empty(t, respReg.GetUserId())
    assert.ErrorContains(t, err, "user already exists")
}

func TestRegister_FailCases(t *testing.T) {
    ctx, st := suite.New(t)

    tests := []struct {
        name        string
        email       string
        password    string
        expectedErr string
    }{
        {
            name:        "Register with Empty Password",
            email:       gofakeit.Email(),
            password:    "",
            expectedErr: "password is required",
        },
        {
            name:        "Register with Empty Email",
            email:       "",
            password:    randomFakePassword(),
            expectedErr: "email is required",
        },
        {
            name:        "Register with Both Empty",
            email:       "",
            password:    "",
            expectedErr: "email is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := st.AuthClient.Register(ctx, &ssov1.RegisterRequest{
                Email:    tt.email,
                Password: tt.password,
            })
            require.Error(t, err)
            require.Contains(t, err.Error(), tt.expectedErr)

        })
    }
}

func TestLogin_FailCases(t *testing.T) {
    ctx, st := suite.New(t)

    tests := []struct {
        name        string
        email       string
        password    string
        appID       int32
        expectedErr string
    }{
        {
            name:        "Login with Empty Password",
            email:       gofakeit.Email(),
            password:    "",
            appID:       appID,
            expectedErr: "password is required",
        },
        {
            name:        "Login with Empty Email",
            email:       "",
            password:    randomFakePassword(),
            appID:       appID,
            expectedErr: "email is required",
        },
        {
            name:        "Login with Both Empty Email and Password",
            email:       "",
            password:    "",
            appID:       appID,
            expectedErr: "email is required",
        },
        {
            name:        "Login with Non-Matching Password",
            email:       gofakeit.Email(),
            password:    randomFakePassword(),
            appID:       appID,
            expectedErr: "invalid email or password",
        },
        {
            name:        "Login without AppID",
            email:       gofakeit.Email(),
            password:    randomFakePassword(),
            appID:       emptyAppID,
            expectedErr: "app_id is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := st.AuthClient.Register(ctx, &ssov1.RegisterRequest{
                Email:    gofakeit.Email(),
                Password: randomFakePassword(),
            })
            require.NoError(t, err)

            _, err = st.AuthClient.Login(ctx, &ssov1.LoginRequest{
                Email:    tt.email,
                Password: tt.password,
                AppId:    tt.appID,
            })
            require.Error(t, err)
            require.Contains(t, err.Error(), tt.expectedErr)
        })
    }
}