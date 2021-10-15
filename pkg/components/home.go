package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

var demoKeys = []GPGKey{
	{
		ID:       "039292",
		FullName: "Isegard Example",
		Email:    "isegard@example.com",
		Private:  true,
	},
	{
		ID:       "838431",
		FullName: "Fred Example",
		Email:    "fred@example.com",
		Private:  true,
		Public:   true,
	},
	{
		ID:       "123456",
		FullName: "Alice Example",
		Email:    "alice@example.com",
		Public:   true,
	},
	{
		ID:       "319312",
		FullName: "Bob Example",
		Email:    "bob@example.com",
		Public:   true,
	},
}

type Home struct {
	app.Compo

	createKeyModalOpen        bool
	importKeyModal            bool
	encryptAndSignModalOpen   bool
	decryptAndVerifyModalOpen bool
}

func (c *Home) Render() app.UI {
	return app.Div().
		Class("pf-c-page").
		Body(
			app.A().
				Class("pf-c-skip-to-content pf-c-button pf-m-primary").
				Href("#gridge-main").
				Body(
					app.Text("Skip to content"),
				),
			&Navbar{},
			app.Main().
				Class("pf-c-page__main").
				ID("gridge-main").
				TabIndex(-1).
				Body(
					app.Section().
						Class("pf-c-page__main-section pf-m-light pf-m-no-padding pf-u-px-sm-on-xl").
						Body(
							&Toolbar{
								OnCreateKey: func() {
									c.createKeyModalOpen = !c.createKeyModalOpen
								},
								OnImportKey: func() {
									c.importKeyModal = !c.importKeyModal
								},

								OnEncryptAndSign: func() {
									c.encryptAndSignModalOpen = !c.encryptAndSignModalOpen
								},
								OnDecryptAndVerify: func() {
									c.decryptAndVerifyModalOpen = !c.decryptAndVerifyModalOpen
								},
							},
						),
					app.Section().
						Class("pf-c-page__main-section pf-m-fill").
						Body(
							&KeyList{
								Keys: demoKeys,
							},
						),
				),
			app.If(
				c.createKeyModalOpen,
				&CreateKeyModal{
					OnSubmit: func(fullName, email, _ string) {
						app.Window().Call("alert", fmt.Sprintf("Created key with full name %v, email %v and a password", fullName, email))

						c.createKeyModalOpen = false
					},
					OnCancel: func() {
						c.createKeyModalOpen = false

						c.Update()
					},
				},
			),
			app.If(
				c.importKeyModal,
				&ImportKeyModal{
					OnSubmit: func(key string) {
						app.Window().Call("alert", fmt.Sprintf("Imported key with contents %v", key))

						c.importKeyModal = false
					},
					OnCancel: func() {
						c.importKeyModal = false

						c.Update()
					},
				},
			),
			app.If(
				c.encryptAndSignModalOpen,
				&EncryptAndSignModal{
					Keys: demoKeys,

					OnSubmit: func(file []byte, publicKeyID, privateKeyID string, createDetachedSignature bool) {
						app.Window().Call("alert", fmt.Sprintf("Encrypted and signed file %v, using public key ID %v and private key ID %v and createDetachedSignature set to %v", file, publicKeyID, privateKeyID, createDetachedSignature))

						c.encryptAndSignModalOpen = false
					},
					OnCancel: func() {
						c.encryptAndSignModalOpen = false

						c.Update()
					},
				},
			),
			app.If(
				c.decryptAndVerifyModalOpen,
				&DecryptAndVerifyModal{
					Keys: demoKeys,

					OnSubmit: func(file []byte, publicKeyID, privateKeyID, detachedSignature string) {
						app.Window().Call("alert", fmt.Sprintf("Decrypted and verified file %v, using public key ID %v, private key ID %v and detached signature %v", file, publicKeyID, privateKeyID, detachedSignature))

						c.decryptAndVerifyModalOpen = false
					},
					OnCancel: func() {
						c.decryptAndVerifyModalOpen = false

						c.Update()
					},
				},
			),
		)
}

func (c *Home) OnAppUpdate(ctx app.Context) {
	if ctx.AppUpdateAvailable() {
		ctx.Reload()
	}
}
