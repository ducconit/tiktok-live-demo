// C ABI wrapper around ttlive::QuickJsSigner so Go (via cgo) can generate
// TikTok X-Bogus / X-Gnarly signatures using a real QuickJS engine (which
// produces deterministic, valid signatures — unlike goja, whose Go-map key
// iteration order breaks the SDK's checksum).
#include "signer_c.h"

#include <cstdlib>
#include <cstring>
#include <string>

#include "qjs_signer.hpp"

struct tiktok_signer {
    ttlive::QuickJsSigner* inner;
    std::string lastError;
};

extern "C" tiktok_signer* tiktok_signer_new(const char* js_dir) {
    if (!js_dir) return nullptr;
    try {
        auto* s = new tiktok_signer;
        s->inner = new ttlive::QuickJsSigner(js_dir);
        return s;
    } catch (const std::exception& e) {
        return nullptr;
    }
}

extern "C" void tiktok_signer_free(tiktok_signer* s) {
    if (!s) return;
    delete s->inner;
    delete s;
}

extern "C" void tiktok_signer_set_cookies(tiktok_signer* s, const char* cookie) {
    if (!s || !s->inner || !cookie) return;
    s->inner->set_cookies(cookie);
}

extern "C" void tiktok_signer_set_user_agent(tiktok_signer* s, const char* ua) {
    if (!s || !s->inner || !ua) return;
    s->inner->set_user_agent(ua);
}

extern "C" char* tiktok_signer_sign(tiktok_signer* s, const char* url) {
    if (!s || !s->inner || !url) return nullptr;
    try {
        std::string result = s->inner->sign(url);
        char* out = static_cast<char*>(std::malloc(result.size() + 1));
        if (!out) return nullptr;
        std::memcpy(out, result.c_str(), result.size() + 1);
        return out;
    } catch (const std::exception& e) {
        s->lastError = e.what();
        return nullptr;
    }
}

extern "C" const char* tiktok_signer_last_error(tiktok_signer* s) {
    if (!s) return "";
    return s->lastError.c_str();
}

extern "C" void tiktok_signer_free_string(char* s) {
    std::free(s);
}
