package secret_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/secret"
)

func TestNewSecret(t *testing.T) {
	t.Parallel()

	t.Run("hide from reflection", func(t *testing.T) {
		t.Parallel()

		secret := secret.New(secretPhrase)
		assert.NotEmpty(t, secret)

		value := reflect.ValueOf(&secret)
		assert.NotContains(t, value.String(), secretPhrase)
		assert.False(t, value.Equal(reflect.ValueOf("******")), "can not guess secret")
		assert.False(t, value.Equal(reflect.ValueOf("secret")), "can not guess secret")
		assert.False(t, value.Equal(reflect.ValueOf(value)), "can not guess secret")

		assert.Panics(t, func() {
			value.FieldByName("secret")
		})
		assert.Panics(t, func() {
			value.Field(0)
		})
		assert.Panics(t, func() {
			reflect.TypeOf(&secret).FieldByName("secret")
		})
	})
}

func TestSecret_Secret(t *testing.T) {
	t.Parallel()

	assert.Equal(t, secretPhrase, secret.New(secretPhrase).Secret())
}

func TestSecret_String(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		secret string
	}{
		"empty":      {""},
		"whitespace": {" "},
		"secret":     {"this-should-be-masked"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			buf := &bytes.Buffer{}
			secret := secret.New(tc.secret)

			fmt.Fprintln(buf, secret)

			assert.Equal(t, "******\n", buf.String(), "should be masked secret")
			// uncomment, to see masking of secrets in action:
			// t.Log(secret)
			// t.Log(buf.String())

			buf.Reset()
			fmt.Fprintln(buf, &secret)
			assert.Equal(t, "******\n", buf.String(), "should be masked secret")

			buf.Reset()
			logger := alog.NewTest(buf)
			logger.Info("msg", slog.Any("secret", secret))

			assert.Contains(t, buf.String(), "******")
			if notEmpty := strings.Trim(tc.secret, " ") != ""; notEmpty {
				assert.NotContains(t, buf.String(), tc.secret, "non empty secret should not contain it's original data")
			}
			// uncomment, to see masking of secrets in action:
			// t.Log(buf.String())
		})
	}
}

func TestSecret_MarshalJSON(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	e := json.NewEncoder(buf)

	err := e.Encode(testStruct{Value: "val", Password: secret.New(secretPhrase)})
	assert.NoError(t, err)
	assert.Equal(t, `{"value":"val","password":"******"}`+"\n", buf.String())
	assert.Contains(t, buf.String(), "******")
	assert.NotContains(t, buf.String(), secretPhrase)
}

func TestSecret_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	obj := testStruct{}
	err := json.Unmarshal([]byte(testRawJSON), &obj)
	assert.NoError(t, err)
	assert.Contains(t, obj.Password.Secret(), secretPhrase)
	assert.Contains(t, obj.Password.String(), "******")
	assert.NotContains(t, obj.Password.String(), secretPhrase)
}

func TestSecret_MarshalText(t *testing.T) {
	t.Parallel()

	secret := secret.New(secretPhrase)

	data, err := secret.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, []byte("******"), data)
}

func TestSecret_UnmarshalText(t *testing.T) {
	t.Parallel()

	secret := secret.New("")
	err := secret.UnmarshalText([]byte(secretPhrase))
	assert.NoError(t, err)
	assert.Equal(t, secretPhrase, secret.Secret())
}

//nolint:errchkjson
func Example_accidentalPrint() {
	secret := secret.New(secretPhrase)
	obj := testStruct{Value: "val", Password: secret}

	fmt.Println(secret)
	fmt.Printf("%+v\n", secret)

	fmt.Printf("%+v\n", obj)
	fmt.Printf("%+v\n", &obj)

	logger := getLogger()
	logger.Info("", slog.Any("secret", secret))

	b, _ := json.Marshal(obj)
	fmt.Println(string(b))

	// Output: ******
	// ******
	// {Value:val Password:******}
	// &{Value:val Password:******}
	// level=INFO msg="" secret=******
	// {"value":"val","password":"******"}
}

func getLogger() *slog.Logger {
	return alog.New(alog.WithHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
		if attr.Key == slog.TimeKey {
			return slog.Attr{}
		}

		return attr
	}})))
}

func Example_unsafeSecretAccess() {
	secret := secret.New(secretPhrase)

	// It is not completely possible to hide the access to the data.
	// If you want stronger guarantees, consider encrypting the data.

	ptrTof := unsafe.Pointer(&secret)

	s := (**string)(ptrTof)
	fmt.Println(**s)

	// Output: this-should-be-secret
}

const secretPhrase = "this-should-be-secret"

type testStruct struct {
	Value    string        `json:"value"`
	Password secret.Secret `json:"password"`
}

var testRawJSON = fmt.Sprintf(`{"value":"val","password":"%s"}`, secretPhrase)
