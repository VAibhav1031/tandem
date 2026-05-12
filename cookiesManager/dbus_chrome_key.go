package cookies

import (
	"bytes"
	"crypto/sha1"
	"strings"

	// "fmt"
	"log"
	"os"

	"github.com/godbus/dbus/v5"
	"golang.org/x/crypto/pbkdf2"
)

func DbusKeyGetter() []byte {

	// Connecting to session bus , because getting the keyring based thing is mostly session based
	// this will give us the some random connection name : 34-402 (everytime)
	conn, err := dbus.SessionBus()
	if err != nil {
		return fallback()
	}
	defer conn.Close()

	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	session := strings.ToLower(os.Getenv("DESKTOP_SESSION"))

	isKDE := strings.Contains(desktop, "kde") || strings.Contains(session, "kde")
	if isKDE {
		return kwalletKey(conn)
	}

	return gnomeKey(conn)
}

func gnomeKey(conn *dbus.Conn) []byte {

	// Proxy Object creation (use for the receive and sending the message across the bus )
	//-----------------Well-Known-Application name  //object-path in that  we want to use
	service := conn.Object("org.freedesktop.secrets", "/org/freedesktop/secrets")

	var sessionPath dbus.ObjectPath
	var placeHolder dbus.Variant
	// Call on the proxy Object is the MEthod call  , and here Secret.Service.OpenSession is the method we are trying to use, Plain (mneans we are not using any encryption for sendinga nd receiving
	// which return the placeholder and sessionPath for that thing in return
	err := service.Call("org.freedesktop.Secret.Service.OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&placeHolder, &sessionPath)
	if err != nil {
		log.Printf("[gnomeKey] : OpenSession Failed = %v", err)
		return fallback()
	}

	// The thing we want to search for
	attrs := map[string]string{
		"xdg:schema": "chrome_libsecret_os_crypt_password_v2",
	}

	var unlocked []dbus.ObjectPath
	var locked []dbus.ObjectPath
	err = service.Call("org.freedesktop.Secret.Service.SearchItems", 0, attrs).Store(&unlocked, &locked)
	if err != nil {
		log.Println("SearchItems failed:", err)
		return fallback()
	}

	if len(unlocked) == 0 && len(locked) == 0 {
		log.Println("[gnomeKey] : no keyring entry found, fallback()..")
		return fallback()
	}

	// grab whichever list has items
	items := unlocked
	if len(items) == 0 {
		items = locked
	}

	// GetSecrets for those item paths
	// return map[ObjectPath]Secret  where Secret is a struct{session, parameters, value, content_type}
	// var secrets map[dbus.ObjectPath][]interface{}
	var secrets map[dbus.ObjectPath][]any
	err = service.Call("org.freedesktop.Secret.Service.GetSecrets", 0, items, sessionPath).
		Store(&secrets)
	if err != nil {
		log.Printf("[gnomeKey] : GetSecrets failed = '%v', fallback()..", err)
		return fallback()
	}

	secret, ok := secrets[items[0]]
	if !ok {
		log.Println("[gnomeKey] : secret not found, fallback()...")
		return fallback()
	}
	value := secret[2].([]byte)

	key := pbkdf2.Key(value, []byte("saltysalt"), 1, 16, sha1.New)
	return key

}

func fallback() []byte {
	return pbkdf2.Key([]byte("peanuts"), []byte("saltysalt"), 1, 16, sha1.New)
}

func kwalletKey(conn *dbus.Conn) []byte {
	for _, service := range []string{"org.kde.kwalletd6", "org.kde.kwalletd5"} {
		path := "/modules/" + strings.TrimPrefix(service, "org.kde.") // idiomatic way : better than checking for the hash prefix and then slicing
		kwallet := conn.Object(service, dbus.ObjectPath(path))

		var handle int32
		err := kwallet.Call("org.kde.KWallet.open", 0, "kdewallet", int64(0), "go-app").
			Store(&handle)
		if err != nil {
			log.Printf("open failed for %s: %v", service, err)
			continue
		}
		log.Printf("opened wallet, handle: %d", handle)
		defer kwallet.Call("org.kde.KWallet.Close", 0, handle, false, "go-app")

		var secret []byte
		err = kwallet.Call("org.kde.KWallet.readEntry", 0, handle, "Chromium Keys", "Chromium Safe Storage", "go-app").Store(&secret)

		if err != nil || len(secret) == 0 {
			log.Printf("readEntry failed: %v", err)
			continue
		}
		log.Printf("secret length: %d raw: %x", len(secret), secret)
		secret = bytes.TrimRight(secret, "\n")
		return pbkdf2.Key(secret, []byte("saltysalt"), 1, 16, sha1.New)
	}
	log.Println("[kwalletKey] : NO key found, fallback..")
	return fallback()

}
