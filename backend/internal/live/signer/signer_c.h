#ifndef TIKTOK_SIGNER_C_H
#define TIKTOK_SIGNER_C_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct tiktok_signer tiktok_signer;

// Create a signer that runs TikTok's webmssdk.js inside QuickJS. js_dir must
// contain hybrid-fake-dom.js + dw-index.js + browser.sg.js + webmssdk.js +
// webmssdk_ex.js + secsdk-lastest.umd.js. Returns NULL on init failure.
tiktok_signer* tiktok_signer_new(const char* js_dir);
void tiktok_signer_free(tiktok_signer* s);

// Inject the browser cookie string (ttwid + msToken) into document.cookie.
void tiktok_signer_set_cookies(tiktok_signer* s, const char* cookie);

// Override the fake DOM's navigator.userAgent.
void tiktok_signer_set_user_agent(tiktok_signer* s, const char* ua);

// Sign a URL; returns a malloc'd string (free with tiktok_signer_free_string)
// or NULL on error.
char* tiktok_signer_sign(tiktok_signer* s, const char* url);
const char* tiktok_signer_last_error(tiktok_signer* s);
void tiktok_signer_free_string(char* s);

#ifdef __cplusplus
}
#endif

#endif
