package harness

// Test-only key pairs for the compatibility suite (generated 2026-11-15).
// The public halves are in configs/<profile>/config.json. Not secrets: never
// use these keys anywhere else.
const (
	rsaPrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCzOacs/6TxWWF2
N7c2p+1qL/K4J559cuMqe3ocnYxzicqZT7JJ15sFtM5i0FtBzYyspjJ3frQcaXe7
hWL/6HmzVO32o5Ea6yPWNHWKOquWGgd07dLLNf18xfab/74fcNTgMxSukC+JXoRX
PXr/pUukygLGimTqgNHbeDxtJSVu5pdIb/1XYino3gO2tu8XllCHjbFtiNXxP5O2
XX9S+yDhTcBcSphle6kMZlgzKVFgMORgjPsM8RLEMUXS0cg6/4Zz0iw6HHgnpHmT
tE4LSHmHnBEbLALr0FeORqCpTXPc6XfgoSGsrzHwZCLOtYqtKBs0OUgO8r4VwudN
xNq24E+ZAgMBAAECggEAAS284rffrtKmqG9fXhWmow8jqoBgM3UA93OObwe+N7hN
Q8rZYlDvrQ7uWIoEP5+JcE07gbuslY9AAVS0PPibbEl/syKKPsITH0KDD2CpZjhP
mj5CJ6H4Q8zNfWMmLfDBBaRwW2GJXn7noec+mtXMdI37jLC2TzB/sNDOnO0knOEW
g8Wz1sqzUy8assXBvHUWN2yVv2O98b5CJJJZscW8Q/YHcCDjtKvlfqNJIBycril0
1O6oKt9I2Emd3VDI9+1y+jtc8/8sY1r0MCapFATLqKsac1XXOvXYgKO6OKAKBXt8
OMlNfspEa05s+Oi8oIARkP0EMeIrDkR7x9zPZ8VKjQKBgQDEzzRSdllPHsQ7HQfe
+VlYQCoUx7lF8thXP0CdJ4jxc0y/8pFyIGEV7mthbtJyImbGeBKtljJMP9M//tKR
6dlhU7YjtyyZlIAoKValYdy98xQmraxPmbjM30fvapz+WfEUtD8H2z+e9+kYRBdt
UVvcHavdbL1QGzn5ut70GXu/PwKBgQDpIJpW1gXQwrfZTfrJ5xq2o5Ju7WPbI7Ps
+X4i1tjrTcE1xsgY3pVtYTqouJ2te3sfdY1AFRDpOKjFuEniJPKGub6ro4eyTqHq
mMkH4xxVJGmNgh1cICdNz4bLK4cXWt+d0JnEO02X+cK+6KMkWGozxNUtg7NjB0go
K8prUYuTJwKBgBWqthjp/JUQTwwu+wHaM7LahYd6IXSI8rZQOV3Tz2i9h9KN8uM8
m1OdxuaYG8UC8ouhBnVj1q+8SxZ9iFMWCp+DZsbx0JAr4BwLyQU2C+ZBj5NehVzS
EJ1Fa5b5BsWdw0PDQDkuMwfeRSQtwuJ/FXzbK2OaKBFcnfpXT1pSP+RjAoGBAOHK
3RFr21gVsCF9YIB7IKlyhhqAPC38R4EnKF8v/grxVUfzjsxfkRTOjhTqMnXbWiXf
If7GKInqgbR2yMCKjoekkaMGESDG/hYNAvBrrH5edQlE05Dt54N6k5BmSUJ6lojw
iG3CDmY61r827QP/ymT3qI2buHwaim219Ci7Y1cTAoGAK8mBmr5rh+YD/8UnEDgd
+oeyJE7LuNwDl122+bfBQJBRSDN0h81AvgeI3ZawV/T2zxrkf9T48ZAw+A6QKRvf
GfnoKiwgMad2NWc2VqwAp6VtQWQ7QdanchUcikGCFKZ9Q0GtoWScrpcvi+b98LNz
wGlwy7NBOEABHM9nO/v2PMQ=
-----END PRIVATE KEY-----
` //nolint:gosec // test-only key
	ecdsaPrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgJYl1IdMrg3HPgcFf
sCRoCwVDNhi+Hn/rrcVDqHSYtiKhRANCAAR80+46yQPn6PIfHI+IxryDUV5agCTF
4NRoroNOwuOdvkrEaaX9op2ubkrnnWQOnj+SKQ93CyMmXhsVPfsJks1C
-----END PRIVATE KEY-----
` //nolint:gosec // test-only key
	ed25519PrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIP9xCP9FK7n2Mc4JYOJd6z6S8WYQFGgefyOizrV/B4w0
-----END PRIVATE KEY-----
` //nolint:gosec // test-only key
)

// Public keys, also used for the alg-confusion case (HS256 signed with the RSA public key).
const (
	RSAPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAszmnLP+k8Vlhdje3Nqft
ai/yuCeefXLjKnt6HJ2Mc4nKmU+ySdebBbTOYtBbQc2MrKYyd360HGl3u4Vi/+h5
s1Tt9qORGusj1jR1ijqrlhoHdO3SyzX9fMX2m/++H3DU4DMUrpAviV6EVz16/6VL
pMoCxopk6oDR23g8bSUlbuaXSG/9V2Ip6N4DtrbvF5ZQh42xbYjV8T+Ttl1/Uvsg
4U3AXEqYZXupDGZYMylRYDDkYIz7DPESxDFF0tHIOv+Gc9IsOhx4J6R5k7ROC0h5
h5wRGywC69BXjkagqU1z3Ol34KEhrK8x8GQizrWKrSgbNDlIDvK+FcLnTcTatuBP
mQIDAQAB
-----END PUBLIC KEY-----
`
	ECDSAPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEfNPuOskD5+jyHxyPiMa8g1FeWoAk
xeDUaK6DTsLjnb5KxGml/aKdrm5K551kDp4/kikPdwsjJl4bFT37CZLNQg==
-----END PUBLIC KEY-----
`
	Ed25519PublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAcbXOdarPyNUP/Zy6pdm7DOLEstRizWHR1YW3q0Ub+y4=
-----END PUBLIC KEY-----
`
)
