package cookies

import (
	"crypto/sha1"
	"github.com/godbus/dbus/v5"
	"golang.org/x/crypto/pbkdf2"
	"log"
)

func DbusKeyGetter() []byte {

	// Connecting to session bus , because getting the keyring based thing is mostly session based
	// this will give us the some random connection name : 34-402
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Proxy Object creation (use for the receive and sending the message across the bus )
	//-----------------Well-Known-Application name  //object-path in that  we want to use
	service := conn.Object("org.freedesktop.secrets", "/org/freedesktop/secrets")

	var sessionPath dbus.ObjectPath
	var placeHolder dbus.Variant
	// Call on the proxy Object is the MEthod call  , and here Secret.Service.OpenSession is the method we are trying to use, Plain (mneans we are not using any encryption for sendinga nd receiving
	// which return the placeholder and sessionPath for that thing in return
	err = service.Call("org.freedesktop.Secret.Service.OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&placeHolder, &sessionPath)

	if err != nil {
		log.Fatal("OpenSession Failed:", err)

	}

	// The thing we want to search for
	attrs := map[string]string{
		"xdg:schema": "chrome_libsecret_os_crypt_password_v2",
	}

	var unlocked []dbus.ObjectPath
	var locked []dbus.ObjectPath
	err = service.Call("org.freedesktop.Secret.Service.SearchItems", 0, attrs).Store(&unlocked, &locked)

	if err != nil {
		log.Fatal("SearchItems failed:", err)
	}

	if len(unlocked) == 0 && len(locked) == 0 {
		log.Println("No keyring entry found, Chromium is using fallback key: peanuts")
		key := pbkdf2.Key([]byte("peanuts"), []byte("saltysalt"), 1, 16, sha1.New)
		return key

		// return []byte{}
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
		log.Fatal("GetSecrets failed:", err)
	}

	secret, ok := secrets[items[0]]
	if !ok {
		log.Fatal("secret not found")
	}
	value := secret[2].([]byte)

	key := pbkdf2.Key(value, []byte("saltysalt"), 1, 16, sha1.New)
	return key

}
