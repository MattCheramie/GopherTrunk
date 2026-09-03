#!/usr/bin/env python3
"""YouTube uploader for the GT-TR-01 manifest (device-flow OAuth, resumable
uploads, thumbnails, captions, playlist, channel branding). Quota-aware:
videos.insert costs 1600 units of the default 10,000/day, so run in batches
(default 6) and re-run daily; a state file skips what's already up.

Commands:
  auth   <client_id> <client_secret> <statedir>     # device flow; prints URL+code, waits
  upload <manifest.json> <statedir> [batch]          # uploads next N pending items
  brand  <statedir> <description.txt> [banner.png]   # channel description + banner
  status <manifest.json> <statedir>

Tokens and upload state live in <statedir> (keep it out of git).
"""
import json, mimetypes, sys, time, urllib.parse, urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SCOPES = "https://www.googleapis.com/auth/youtube.upload https://www.googleapis.com/auth/youtube.force-ssl"


def http(url, data=None, headers=None, method=None, raw=False):
    req = urllib.request.Request(url, data=data, headers=headers or {}, method=method)
    try:
        with urllib.request.urlopen(req, timeout=600) as r:
            body = r.read()
            return r.status, dict(r.headers), (body if raw else json.loads(body or b"{}"))
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="replace")
        return e.code, dict(e.headers), body


def load(p):
    return json.loads(Path(p).read_text()) if Path(p).exists() else {}


def save(p, obj):
    Path(p).parent.mkdir(parents=True, exist_ok=True)
    Path(p).write_text(json.dumps(obj, indent=1))


def auth(client_id, client_secret, statedir):
    st, _, dev = http("https://oauth2.googleapis.com/device/code",
                      urllib.parse.urlencode({"client_id": client_id, "scope": SCOPES}).encode(),
                      {"Content-Type": "application/x-www-form-urlencoded"})
    if st != 200:
        raise SystemExit(f"device code failed: {st} {dev}")
    print(f"\n==> On your phone/computer, open  {dev['verification_url']}  and enter code:  {dev['user_code']}\n")
    print(f"(waiting up to {dev['expires_in']}s for approval)")
    while True:
        time.sleep(dev.get("interval", 5))
        st, _, tok = http("https://oauth2.googleapis.com/token",
                          urllib.parse.urlencode({
                              "client_id": client_id, "client_secret": client_secret,
                              "device_code": dev["device_code"],
                              "grant_type": "urn:ietf:params:oauth:grant-type:device_code"}).encode(),
                          {"Content-Type": "application/x-www-form-urlencoded"})
        if st == 200:
            tok["client_id"] = client_id; tok["client_secret"] = client_secret
            tok["obtained_at"] = time.time()
            save(Path(statedir) / "token.json", tok)
            print("authorized — token stored")
            return
        err = json.loads(tok)["error"] if isinstance(tok, str) else tok.get("error")
        if err == "authorization_pending":
            continue
        if err == "slow_down":
            time.sleep(5); continue
        raise SystemExit(f"auth failed: {err}")


def access_token(statedir):
    tokp = Path(statedir) / "token.json"
    tok = load(tokp)
    if not tok:
        raise SystemExit("no token — run auth first")
    if time.time() - tok.get("obtained_at", 0) > tok.get("expires_in", 3600) - 120:
        st, _, new = http("https://oauth2.googleapis.com/token",
                          urllib.parse.urlencode({
                              "client_id": tok["client_id"], "client_secret": tok["client_secret"],
                              "refresh_token": tok["refresh_token"],
                              "grant_type": "refresh_token"}).encode(),
                          {"Content-Type": "application/x-www-form-urlencoded"})
        if st != 200:
            raise SystemExit(f"token refresh failed: {st} {new}")
        tok.update(new); tok["obtained_at"] = time.time()
        save(tokp, tok)
    return tok["access_token"]


def api(tokendir, method, url, body=None, raw_body=None, ctype="application/json"):
    at = access_token(tokendir)
    data = raw_body if raw_body is not None else (json.dumps(body).encode() if body is not None else None)
    return http(url, data, {"Authorization": f"Bearer {at}", "Content-Type": ctype}, method=method)


def upload_video(statedir, item):
    meta = {"snippet": {"title": item["title"], "description": item["description"],
                        "tags": item["tags"], "categoryId": item.get("categoryId", "28")},
            "status": {"privacyStatus": "private", "publishAt": item["publishAt"],
                       "selfDeclaredMadeForKids": False}}
    fp = Path(item["file"]); size = fp.stat().st_size
    at = access_token(statedir)
    st, hdrs, body = http(
        "https://www.googleapis.com/upload/youtube/v3/videos?uploadType=resumable&part=snippet,status",
        json.dumps(meta).encode(),
        {"Authorization": f"Bearer {at}", "Content-Type": "application/json",
         "X-Upload-Content-Length": str(size), "X-Upload-Content-Type": "video/mp4"})
    if st != 200:
        raise RuntimeError(f"resumable init failed: {st} {body}")
    loc = hdrs.get("Location") or hdrs.get("location")
    st, _, body = http(loc, fp.read_bytes(),
                       {"Authorization": f"Bearer {access_token(statedir)}",
                        "Content-Type": "video/mp4", "Content-Length": str(size)}, method="PUT")
    if st not in (200, 201):
        raise RuntimeError(f"upload failed: {st} {body}")
    vid = body["id"]
    # thumbnail
    if item.get("thumb"):
        tp = ROOT / item["thumb"]
        if tp.exists():
            st, _, r = api(statedir, "POST",
                           f"https://www.googleapis.com/upload/youtube/v3/thumbnails/set?videoId={vid}",
                           raw_body=tp.read_bytes(), ctype="image/png")
            if st not in (200, 201):
                print(f"    thumbnail failed ({st}): {str(r)[:200]}")
    # captions
    if item.get("srt") and Path(item["srt"]).exists():
        boundary = "gtb0undary"
        capmeta = json.dumps({"snippet": {"videoId": vid, "language": "en", "name": "English"}})
        srt = Path(item["srt"]).read_bytes()
        mp = (f"--{boundary}\r\nContent-Type: application/json\r\n\r\n{capmeta}\r\n"
              f"--{boundary}\r\nContent-Type: application/octet-stream\r\n\r\n").encode() + srt + f"\r\n--{boundary}--".encode()
        st, _, r = api(statedir, "POST",
                       "https://www.googleapis.com/upload/youtube/v3/captions?uploadType=multipart&part=snippet",
                       raw_body=mp, ctype=f"multipart/related; boundary={boundary}")
        if st not in (200, 201):
            print(f"    captions failed ({st}): {str(r)[:200]}")
    return vid


def ensure_playlist(statedir, state, manifest):
    if state.get("playlistId"):
        return state["playlistId"]
    st, _, r = api(statedir, "POST",
                   "https://www.googleapis.com/youtube/v3/playlists?part=snippet,status",
                   {"snippet": {"title": manifest["playlistTitle"],
                                "description": manifest["playlistDescription"]},
                    "status": {"privacyStatus": "public"}})
    if st not in (200, 201):
        raise RuntimeError(f"playlist create failed: {st} {r}")
    state["playlistId"] = r["id"]
    return r["id"]


def upload(manifest_path, statedir, batch=6):
    manifest = load(manifest_path)
    statep = Path(statedir) / "uploaded.json"
    state = load(statep)
    done = state.setdefault("videos", {})
    n = 0
    for item in manifest["items"]:
        if item["key"] in done or n >= batch:
            continue
        print(f"uploading {item['key']}: {item['title'][:60]} "
              f"({Path(item['file']).stat().st_size // 1048576} MiB)")
        vid = upload_video(statedir, item)
        done[item["key"]] = vid
        save(statep, state)
        print(f"  -> https://youtu.be/{vid}  (private, publishes {item['publishAt']})")
        if item.get("playlist"):
            pl = ensure_playlist(statedir, state, manifest)
            save(statep, state)
            st, _, r = api(statedir, "POST",
                           "https://www.googleapis.com/youtube/v3/playlistItems?part=snippet",
                           {"snippet": {"playlistId": pl,
                                        "resourceId": {"kind": "youtube#video", "videoId": vid}}})
            if st not in (200, 201):
                print(f"    playlist add failed ({st}): {str(r)[:200]}")
        n += 1
    pend = [i["key"] for i in manifest["items"] if i["key"] not in done]
    print(f"\nbatch done: {n} uploaded this run, {len(done)} total, pending: {pend or 'none'}")


def brand(statedir, desc_path, banner=None):
    st, _, ch = api(statedir, "GET",
                    "https://www.googleapis.com/youtube/v3/channels?part=id,brandingSettings&mine=true")
    if st != 200 or not ch.get("items"):
        raise SystemExit(f"channels.list failed: {st} {str(ch)[:300]}")
    c = ch["items"][0]
    bs = c.get("brandingSettings", {})
    bs.setdefault("channel", {})["description"] = Path(desc_path).read_text()
    bs["channel"]["keywords"] = "SDR \"trunked radio\" P25 DMR TETRA scanner GopherTrunk"
    if banner:
        st, _, r = api(statedir, "POST",
                       "https://www.googleapis.com/upload/youtube/v3/channelBanners/insert",
                       raw_body=Path(banner).read_bytes(), ctype="image/png")
        if st in (200, 201):
            bs.setdefault("image", {})["bannerExternalUrl"] = r["url"]
        else:
            print(f"banner upload failed ({st}): {str(r)[:200]}")
    st, _, r = api(statedir, "PUT",
                   "https://www.googleapis.com/youtube/v3/channels?part=brandingSettings",
                   {"id": c["id"], "brandingSettings": bs})
    print("branding update:", st if st == 200 else f"{st} {str(r)[:300]}")


def status(manifest_path, statedir):
    manifest = load(manifest_path)
    done = load(Path(statedir) / "uploaded.json").get("videos", {})
    for i in manifest["items"]:
        mark = done.get(i["key"], "pending")
        print(f"{i['key']:7s} {mark:15s} {i['publishAt']}  {i['title'][:58]}")


if __name__ == "__main__":
    cmd = sys.argv[1]
    if cmd == "auth":
        auth(sys.argv[2], sys.argv[3], sys.argv[4])
    elif cmd == "upload":
        upload(sys.argv[2], sys.argv[3], int(sys.argv[4]) if len(sys.argv) > 4 else 6)
    elif cmd == "brand":
        brand(sys.argv[2], sys.argv[3], sys.argv[4] if len(sys.argv) > 4 else None)
    elif cmd == "status":
        status(sys.argv[2], sys.argv[3])
