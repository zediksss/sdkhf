package controller

import (
	"html/template"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"h-ui/model/constant"
	"h-ui/service"
)

type subscribePageView struct {
	Title       string
	Username    string
	Status      string
	StatusClass string
	ExpireDate  string
	Traffic     string
	RawURL      string
	LogoURL     string
	FontURL     string
	SupportURL  string
}

func Hysteria2SubscribeAsset(c *gin.Context) {
	fileName := c.Param("file")
	var localPath string
	var contentType string
	switch fileName {
	case "logo.png":
		localPath = "logo.png"
		contentType = "image/png"
	case "font.ttf":
		localPath = "font.ttf"
		contentType = "font/ttf"
	default:
		c.String(http.StatusNotFound, "404")
		return
	}

	absPath, err := filepath.Abs(localPath)
	if err != nil || !fileExists(absPath) {
		c.String(http.StatusNotFound, "404")
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Content-Type", contentType)
	c.File(absPath)
}

func renderHysteria2SubscribePage(c *gin.Context, subToken string, host string) {
	pageData, err := service.Hysteria2SubscribePageByToken(subToken)
	if err != nil {
		c.String(http.StatusNotFound, "subscription not found")
		return
	}

	account := pageData.Account
	now := time.Now().UnixMilli()
	used := int64(0)
	if account.Download != nil {
		used += *account.Download
	}
	if account.Upload != nil {
		used += *account.Upload
	}
	quota := int64(-1)
	if account.Quota != nil {
		quota = *account.Quota
	}
	expireTime := int64(0)
	if account.ExpireTime != nil {
		expireTime = *account.ExpireTime
	}

	status := "Активна"
	statusClass := "is-active"
	if (quota >= 0 && used >= quota) || expireTime <= now {
		status = "Неактивна"
		statusClass = "is-muted"
	}

	basePath := subscribeBasePath(c.Request.URL.Path)
	rawURL := requestScheme(c) + "://" + host + c.Request.URL.Path + "?raw=1"
	view := subscribePageView{
		Title:       pageData.ProfileName,
		Username:    valueOrEmpty(account.Username),
		Status:      status,
		StatusClass: statusClass,
		ExpireDate:  formatDateRu(expireTime),
		Traffic:     formatTrafficRu(used, quota),
		RawURL:      rawURL,
		LogoURL:     basePath + "/sub-assets/logo.png",
		FontURL:     basePath + "/sub-assets/font.ttf",
		SupportURL:  "https://t.me/keysforuuu",
	}

	c.Header("Cache-Control", "no-store")
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := subscribePageTemplate.Execute(c.Writer, view); err != nil {
		c.String(http.StatusInternalServerError, "render error")
	}
}

func isBrowserUserAgent(userAgent string) bool {
	return strings.Contains(userAgent, "mozilla") ||
		strings.Contains(userAgent, "chrome") ||
		strings.Contains(userAgent, "safari") ||
		strings.Contains(userAgent, "firefox") ||
		strings.Contains(userAgent, "edg") ||
		strings.Contains(userAgent, "opera")
}

func isKnownSubscribeClient(userAgent string) bool {
	return strings.Contains(userAgent, constant.Shadowrocket) ||
		strings.Contains(userAgent, constant.Happ) ||
		strings.Contains(userAgent, constant.Clash) ||
		strings.Contains(userAgent, constant.V2rayN) ||
		strings.Contains(userAgent, constant.NekoBox)
}

func requestScheme(c *gin.Context) string {
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return strings.Split(proto, ",")[0]
	}
	if c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

func subscribeBasePath(path string) string {
	index := strings.LastIndex(path, "/sub/")
	if index <= 0 {
		return ""
	}
	return path[:index]
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func formatDateRu(timestamp int64) string {
	if timestamp <= 0 {
		return "-"
	}
	months := []string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}
	date := time.UnixMilli(timestamp)
	return date.Format("2 ") + months[int(date.Month())-1] + date.Format(", 2006")
}

func formatTrafficRu(used int64, quota int64) string {
	if quota < 0 {
		return formatBytesShortRu(used) + "/Безлимит"
	}
	return formatBytesShortRu(used) + "/" + formatBytesShortRu(quota)
}

func formatBytesShortRu(bytes int64) string {
	if bytes <= 0 {
		return "0Б"
	}
	units := []string{"Б", "КБ", "МБ", "ГБ", "ТБ", "ПБ"}
	value := float64(bytes)
	index := 0
	for value >= 1024 && index < len(units)-1 {
		value /= 1024
		index++
	}
	if math.Abs(value-math.Round(value)) < 0.05 {
		return strings.ReplaceAll(formatFloat(value, 0), ".", ",") + units[index]
	}
	return strings.ReplaceAll(formatFloat(value, 1), ".", ",") + units[index]
}

func formatFloat(value float64, precision int) string {
	if precision == 0 {
		return strconv.FormatFloat(value, 'f', 0, 64)
	}
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', precision, 64), "0"), ".")
}

var subscribePageTemplate = template.Must(template.New("subscribe-page").Parse(`<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} - подписка</title>
  <style>
    @font-face{font-family:SubFont;src:url('{{.FontURL}}') format('truetype');font-display:swap}
    *{box-sizing:border-box}
    body{margin:0;min-height:100vh;background:#17181c;color:#f4f4f5;font-family:SubFont,Inter,Arial,sans-serif;letter-spacing:-.03em}
    .page{width:min(560px,calc(100vw - 32px));margin:20px auto 28px}
    .brand{display:flex;align-items:center;gap:10px;margin:0 0 18px 2px;font-weight:800;font-size:23px;line-height:1}
    .brand img{width:31px;height:31px;object-fit:contain}
    .panel{background:#24272c;border-radius:11px;padding:16px 18px;margin-bottom:17px;box-shadow:0 1px 0 rgba(255,255,255,.02) inset}
    h1,h2{margin:0;font-weight:900;line-height:1.05}
    h1{font-size:22px;margin-bottom:17px}
    h2{font-size:22px}
    .info-grid{display:grid;grid-template-columns:1fr 1fr;gap:6px}
    .info-card{min-height:43px;border:1px solid rgba(255,255,255,.11);border-radius:7px;padding:7px 10px;background:linear-gradient(100deg,#30313b,#2c2f35);box-shadow:inset 0 1px 1px rgba(255,255,255,.07)}
    .info-card.good{background:linear-gradient(100deg,#25342f,#303631)}
    .info-card.warn{background:linear-gradient(100deg,#312b2d,#373331)}
    .label{display:flex;align-items:center;gap:4px;color:#ddd;font-size:9px;line-height:1.1}
    .value{display:block;margin-top:4px;color:#fff;font-size:13px;font-weight:850;line-height:1.05;overflow-wrap:anywhere}
    .is-muted{color:#c9c9cc}
    svg{width:13px;height:13px;display:block;flex:0 0 auto}
    .head{display:flex;align-items:center;justify-content:space-between;gap:14px;margin-bottom:14px}
    .device{height:24px;border:0;border-radius:10px;background:#1e2024;color:#aeb0b6;padding:0 28px 0 11px;font:inherit;font-size:12px;letter-spacing:-.03em;outline:none}
    .steps{display:flex;flex-direction:column;gap:7px}
    .step{display:grid;grid-template-columns:38px 1fr auto;align-items:center;gap:10px;min-height:56px;padding:10px 16px;background:#1f2226;border-radius:10px}
    .bubble{width:29px;height:29px;border-radius:50%;display:grid;place-items:center;background:#252a2e;color:#888}
    .step-title{font-size:14px;font-weight:850;line-height:1.1;margin-bottom:4px}
    .step-text{font-size:12px;color:#8f9197;line-height:1.18;max-width:390px}
    .btn{display:inline-flex;align-items:center;justify-content:center;height:18px;min-width:84px;border-radius:6px;border:1px solid #46516b;background:linear-gradient(#303748,#252b3a);color:#fff;text-decoration:none;font-size:10px;font-weight:750;padding:0 12px;letter-spacing:-.03em;cursor:pointer}
    .support{display:grid;grid-template-columns:1fr auto;align-items:center;gap:14px;padding:14px 26px 12px 16px}
    .support h2{font-size:23px}
    .support p{margin:3px 0 0;color:#8f9197;font-size:12px;line-height:1.25}
    .copy-row{display:flex;gap:8px;margin-top:10px}
    .copy-row input{width:100%;min-width:0;height:28px;border:1px solid rgba(255,255,255,.08);border-radius:7px;background:#1f2226;color:#aaa;padding:0 9px;font:inherit;font-size:11px;letter-spacing:0}
    @media (max-width:620px){
      .page{width:min(100% - 22px,560px);margin:14px auto 22px}
      .brand{font-size:21px}
      .panel{padding:15px 14px;border-radius:10px}
      .info-grid{grid-template-columns:1fr}
      .step{grid-template-columns:34px 1fr;align-items:start}
      .step .btn{grid-column:2;justify-self:start;margin-top:2px}
      .support{grid-template-columns:1fr;padding:14px}
      .support .btn{justify-self:start}
    }
  </style>
</head>
<body>
  <main class="page">
    <div class="brand"><img src="{{.LogoURL}}" alt=""><span>ЕСЕНИН ВРН</span></div>

    <section class="panel">
      <h1>Подписка {{.Username}}</h1>
      <div class="info-grid">
        <div class="info-card">
          <span class="label"><svg viewBox="0 0 24 24" fill="none"><path d="M20 21a8 8 0 0 0-16 0" stroke="#f4c95d" stroke-width="2" stroke-linecap="round"/><circle cx="12" cy="7" r="4" stroke="#f4c95d" stroke-width="2"/></svg>имя пользователя</span>
          <span class="value">{{.Username}}</span>
        </div>
        <div class="info-card good">
          <span class="label"><svg viewBox="0 0 24 24" fill="none"><path d="m5 12 4 4L19 6" stroke="#2cf56f" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/></svg>статус</span>
          <span class="value {{.StatusClass}}">{{.Status}}</span>
        </div>
        <div class="info-card warn">
          <span class="label"><svg viewBox="0 0 24 24" fill="none"><path d="M7 3v4M17 3v4M4 9h16M6 5h12a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Z" stroke="#ffd19a" stroke-width="2" stroke-linecap="round"/></svg>истекает</span>
          <span class="value">{{.ExpireDate}}</span>
        </div>
        <div class="info-card">
          <span class="label"><svg viewBox="0 0 24 24" fill="none"><path d="M5 13a10 10 0 0 1 14 0M8.5 16.5a5 5 0 0 1 7 0M12 20h.01" stroke="#4c8dff" stroke-width="2.5" stroke-linecap="round"/></svg>трафик</span>
          <span class="value">{{.Traffic}}</span>
        </div>
      </div>
      <div class="copy-row">
        <input id="subUrl" value="{{.RawURL}}" readonly aria-label="Ссылка подписки">
        <button class="btn" id="copyBtn" type="button">Скопировать</button>
      </div>
    </section>

    <section class="panel">
      <div class="head">
        <h2>Инструкция по установке</h2>
        <select class="device" id="device">
          <option value="ios">ios</option>
          <option value="android">android</option>
          <option value="windows">windows</option>
        </select>
      </div>
      <div class="steps">
        <div class="step">
          <div class="bubble"><svg viewBox="0 0 24 24" fill="none"><path d="M12 3v12m0 0 4-4m-4 4-4-4M5 21h14" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg></div>
          <div><div class="step-title" id="s1t">Установите приложение HAPP</div><div class="step-text" id="s1d">Откройте страницу в AppStore и установите приложение Happ.</div></div>
          <a class="btn" id="installBtn" href="https://apps.apple.com/search?term=HAPP" target="_blank" rel="noreferrer">Установить</a>
        </div>
        <div class="step">
          <div class="bubble"><svg viewBox="0 0 24 24" fill="none"><path d="M8 12h8M12 8v8M6 4h12a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg></div>
          <div><div class="step-title">Добавьте подписку в приложение</div><div class="step-text" id="s2d">Скопируйте ссылку на оплаченную подписку в боте. Перейдите в приложение HAPP, нажмите на + в правом верхнем углу. Выберите пункт "Вставить из буфера обмена".</div></div>
        </div>
        <div class="step">
          <div class="bubble"><svg viewBox="0 0 24 24" fill="none"><path d="M13 2 4 14h7l-1 8 10-13h-7l0-7Z" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/></svg></div>
          <div><div class="step-title">Подключитесь к ускорителю</div><div class="step-text">В HAPP в списке подписок нажмите на ЕСЕНИН ВРН, а далее нажмите на большую кнопку включения.</div></div>
        </div>
      </div>
    </section>

    <section class="panel support">
      <div><h2>Возникли проблемы?</h2><p>Наша поддержка на связи 24/7. В случае трудностей напишите нам.</p></div>
      <a class="btn" href="{{.SupportURL}}" target="_blank" rel="noreferrer">Помощь</a>
    </section>
  </main>
  <script>
    const data = {
      ios: {label:'ios', href:'https://apps.apple.com/search?term=HAPP', s1:'Откройте страницу в AppStore и установите приложение Happ.', s2:'Скопируйте ссылку на оплаченную подписку в боте. Перейдите в приложение HAPP, нажмите на + в правом верхнем углу. Выберите пункт "Вставить из буфера обмена".'},
      android: {label:'android', href:'https://play.google.com/store/search?q=HAPP&c=apps', s1:'Откройте страницу в Google Play и установите приложение Happ.', s2:'Скопируйте ссылку на оплаченную подписку в боте. Перейдите в приложение HAPP, нажмите на + в правом верхнем углу. Выберите пункт "Вставить из буфера обмена".'},
      windows: {label:'windows', href:'https://github.com/Happ-proxy/happ-desktop/releases', s1:'Установите приложение HAPP по ссылке.', s2:'Откройте приложение, скопируйте ссылку из бота. Перейдите в HAPP и нажмите CTRL+V.'}
    };
    const device = document.getElementById('device');
    const installBtn = document.getElementById('installBtn');
    const s1d = document.getElementById('s1d');
    const s2d = document.getElementById('s2d');
    const ua = navigator.userAgent.toLowerCase();
    if (ua.includes('android')) device.value = 'android';
    else if (ua.includes('windows')) device.value = 'windows';
    const update = () => { const item = data[device.value]; installBtn.href = item.href; s1d.textContent = item.s1; s2d.textContent = item.s2; };
    device.addEventListener('change', update);
    update();
    document.getElementById('copyBtn').addEventListener('click', async () => {
      const input = document.getElementById('subUrl');
      try { await navigator.clipboard.writeText(input.value); } catch(e) { input.select(); document.execCommand('copy'); }
      document.getElementById('copyBtn').textContent = 'Скопировано';
      setTimeout(() => document.getElementById('copyBtn').textContent = 'Скопировать', 1200);
    });
  </script>
</body>
</html>`))
