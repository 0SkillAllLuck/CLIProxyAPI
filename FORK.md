# 0SkillAllLuck/CLIProxyAPI

Upstream [router-for-me/CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)
at a release tag, plus the patches below. `patched` is the branch that builds;
every push to it publishes `ghcr.io/0skillallluck/cliproxyapi:v<upstream>.<run>`
(private, linux/amd64) via `.github/workflows/fork-image.yml`.

## Patches

One commit each, oldest first. Drop a commit once upstream ships the fix.

| Commit | Why |
| --- | --- |
| fix(xai): resolve token endpoint via discovery before refresh (#6890) | xAI retired the stored token URL; old Grok logins can't refresh |
| fix(executor): normalize Codex multi-agent v2 agent_message items for Kimi Responses (#5914) | Kimi 400s on Codex subagent history |
| fix(antigravity): synthesize missing functionCall IDs for Claude models (#7784) | the Antigravity CLI drops tool-call IDs; Claude via Antigravity rejects them |
| feat(access): scope client API keys to providers (#6568) | per-key provider allowlist (`client.key-scopes`) |
| fix(access): scope Muse Code and AI Studio relay routes | ours: closes two routes #6568 leaves open |
| fix(access): fail closed on unmatched keys and empty scopes | ours: #6568 lets such keys use every provider |

Muse needs no patch: v8.1.2 already serves the Muse CLI endpoints.

## Key scopes

Provider names are executor identifiers: `claude`, `codex`, `antigravity`,
`meta` (Muse), `xai` (Grok), `kimi` (kimi.com login), `kimi-ai` (kimi.ai login),
`aistudio` (the `/v1/ws` relay), and openai-compatibility entry names. Don't name
an openai-compatibility entry after any provider name above or an alias
(`anthropic`, `openai`, `chatgpt`, `muse`, `grok`): scoping to it would also grant
that subscription provider.
Never start it in Home mode (`-home-jwt` or `HOME_JWT`): key scopes do not apply
there.

Once any `client.key-scopes` entry exists, scoping fails closed: a request
without a key, a key that matches no `key-prefix`, or one whose entry lists no
provider may use no provider at all, and so may a call that lost its request.
So even an emptied `access.api-keys` or a disabled `ws-auth` doesn't open the
proxy while scopes are configured.

Entries under `api-keys.claude` count as `claude`, so put GLM under
`api-keys.openai-compatibility`. Keep `oauth.providers.aistudio.ws-auth: true`
(the default), or `/v1/ws` takes connections without a key. `/v1/tunnel` is not
scoped; it serves no provider, and each key gets its own realm.

```yaml
config-version: 8
access:
  api-keys: ["sk-personal-...", "sk-work-...", "sk-untrusted-..."]
client:
  key-scopes:
    - key-prefix: "sk-untrusted-"
      providers: ["kimi", "kimi-ai", "glm"]
    - key-prefix: "sk-personal-"
      providers: ["claude", "codex", "antigravity", "meta", "xai"]
    - key-prefix: "sk-work-"
      providers: ["claude", "codex", "antigravity", "meta", "xai"]
```

## Rebase onto a new upstream release

```bash
git fetch upstream --tags
git rebase --onto vNEW vOLD patched   # fix conflicts, drop patches upstream merged
# Upstream workflows are disabled one by one, so a new one runs in the fork as
# soon as the push below triggers it. Stop if this lists anything: delete those
# files in the fork CI commit, push, then disable them in the fork's settings.
git diff --name-only --diff-filter=AR vOLD vNEW -- .github/workflows
git push origin vNEW                  # describe needs the base tag in the fork
git push --force-with-lease origin patched
# Then disable everything active except fork-image.yml:
gh api repos/0SkillAllLuck/CLIProxyAPI/actions/workflows \
  --jq '.workflows[] | select(.state == "active") | .path'
```

The infra repo only accepts four-part fork tags (`v8.1.2.N`) from this image, so
an image an upstream workflow might publish is never offered as an update.
