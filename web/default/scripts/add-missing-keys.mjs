import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return `${JSON.stringify(obj, null, 2)}\n`
}

const newKeys = {
  en: {
    'Upload status': 'Upload status',
    Uploaded: 'Uploaded',
    'Human verification is loading': 'Human verification is loading',
    'Human verification is taking longer than expected':
      'Human verification is taking longer than expected',
    'If it does not appear for a long time, refresh the page and try again.':
      'If it does not appear for a long time, refresh the page and try again.',
    'Please wait a moment...': 'Please wait a moment...',
    'Proxy Video Content': 'Proxy Video Content',
    'Redirect to the HTTPS URL returned by upstream task details':
      'Redirect to the HTTPS URL returned by upstream task details',
    'Stream video content through this server for reliable preview and download':
      'Stream video content through this server for reliable preview and download',
    'HTTP Redirect': 'HTTP Redirect',
    'Require upstream content requests to return a redirect; non-redirect responses fail without proxy fallback':
      'Require upstream content requests to return a redirect; non-redirect responses fail without proxy fallback',
    'Reverse Proxy': 'Reverse Proxy',
    'Stream video content through this server':
      'Stream video content through this server',
    'Video Content Delivery': 'Video Content Delivery',
    '(fixed fee + per-second price × duration) × output count':
      '(fixed fee + per-second price × duration) × output count',
    Actual: 'Actual',
    'Billing method': 'Billing method',
    'Choose image quality, image size tier, or exact image size for images; choose video resolution tier for videos. Enter values exactly as supported by the upstream model.':
      'Choose image quality, image size tier, or exact image size for images; choose video resolution tier for videos. Enter values exactly as supported by the upstream model.',
    'Condition value': 'Condition value',
    Each: 'Each',
    'Enter a provider-supported image quality value, for example high':
      'Enter a provider-supported image quality value, for example high',
    'Enter a provider-supported quality value, for example high':
      'Enter a provider-supported quality value, for example high',
    Estimated: 'Estimated',
    'Fixed fee': 'Fixed fee',
    'Fixed fee + per second': 'Fixed fee + per second',
    'Fixed fee plus per second': 'Fixed fee plus per second',
    'Generated expression': 'Generated expression',
    'Frimodel mapping: 1024x1024 is 1K; 1536x1024, 1024x1536, auto, and missing size are 2K. Larger custom sizes are classified by their longest edge.':
      'Frimodel mapping: 1024x1024 is 1K; 1536x1024, 1024x1536, auto, and missing size are 2K. Larger custom sizes are classified by their longest edge.',
    'How media pricing works': 'How media pricing works',
    'Image price by size': 'Image price by size',
    'Image quality': 'Image quality',
    'Image size': 'Image size',
    'Image size tier': 'Image size tier',
    'Media billing dimensions': 'Media billing dimensions',
    'Media editor': 'Media editor',
    'Media generation': 'Media generation',
    'Media price': 'Media price',
    'Media tiers use validated dimensions. The last tier is always the fallback.':
      'Media tiers use validated dimensions. The last tier is always the fallback.',
    'No condition value required': 'No condition value required',
    'Per image': 'image',
    'Per output': 'output',
    'Per second': 'Per second',
    'Per video': 'video',
    'Per-second price': 'Per-second price',
    'per-second price × duration × output count':
      'per-second price × duration × output count',
    'Price preview shows the configured rate. Actual charges use the validated output count and video duration.':
      'Price preview shows the configured rate. Actual charges use the validated output count and video duration.',
    'Price preview': 'Price preview',
    Quality: 'Quality',
    'Resolution tier': 'Resolution tier',
    second: 'second',
    'Settlement delta': 'Settlement delta',
    'Tier condition': 'Tier condition',
    'Tiers are checked from top to bottom. The first matching tier sets the price.':
      'Tiers are checked from top to bottom. The first matching tier sets the price.',
    'The last tier applies when no earlier condition matches, so it does not need a condition value.':
      'The last tier applies when no earlier condition matches, so it does not need a condition value.',
    'Unit price': 'Unit price',
    Units: 'Units',
    'unit price × output count': 'unit price × output count',
    'Use quality or image size for images, and resolution for videos. Enter values exactly as supported by the upstream model.':
      'Use quality or image size for images, and resolution for videos. Enter values exactly as supported by the upstream model.',
    'Video fixed fee plus duration': 'Video fixed fee plus duration',
    'Video price by resolution': 'Video price by resolution',
    'Video resolution tier': 'Video resolution tier',
  },
  zh: {
    'Upload status': '上传状态',
    Uploaded: '已上传',
    'Human verification is loading': '人机验证加载中',
    'Human verification is taking longer than expected': '人机验证加载时间较长',
    'If it does not appear for a long time, refresh the page and try again.':
      '如果长时间没有出现，请刷新页面后重试。',
    'Please wait a moment...': '请稍等...',
    'Proxy Video Content': '代理视频内容',
    'Redirect to the HTTPS URL returned by upstream task details':
      '重定向到上游任务详情返回的 HTTPS 地址',
    'Stream video content through this server for reliable preview and download':
      '由本服务器转发视频内容，以获得稳定的预览和下载体验',
    'HTTP Redirect': 'HTTP 重定向',
    'Require upstream content requests to return a redirect; non-redirect responses fail without proxy fallback':
      '要求上游内容请求返回重定向；非重定向响应直接报错，不回退代理',
    'Reverse Proxy': '反向代理',
    'Stream video content through this server': '由本服务器转发视频内容',
    'Video Content Delivery': '视频内容交付方式',
    '(fixed fee + per-second price × duration) × output count':
      '（固定费用 + 每秒价格 × 时长）× 输出数量',
    Actual: '实际',
    'Billing method': '计费方式',
    'Choose image quality, image size tier, or exact image size for images; choose video resolution tier for videos. Enter values exactly as supported by the upstream model.':
      '图片可选择图片质量、图片尺寸档位或具体图片尺寸；视频请选择视频分辨率档位。条件值应与上游模型支持的值完全一致。',
    'Condition value': '条件值',
    Each: '每个',
    'Enter a provider-supported image quality value, for example high':
      '填写上游支持的图片质量值，例如 high',
    'Enter a provider-supported quality value, for example high':
      '填写上游支持的质量值，例如 high',
    Estimated: '预估',
    'Fixed fee': '固定费用',
    'Fixed fee + per second': '固定费用 + 每秒',
    'Fixed fee plus per second': '固定费用加每秒费用',
    'Generated expression': '生成的表达式',
    'Frimodel mapping: 1024x1024 is 1K; 1536x1024, 1024x1536, auto, and missing size are 2K. Larger custom sizes are classified by their longest edge.':
      'Frimodel 映射：1024x1024 为 1K；1536x1024、1024x1536、auto 和未填写尺寸均为 2K。更大的自定义尺寸按最长边归档。',
    'How media pricing works': '媒体价格如何计算',
    'Image price by size': '按尺寸计算图片价格',
    'Image quality': '图片质量',
    'Image size': '图片尺寸',
    'Image size tier': '图片尺寸档位',
    'Media billing dimensions': '媒体计费维度',
    'Media editor': '媒体编辑器',
    'Media generation': '媒体生成',
    'Media price': '媒体价格',
    'Media tiers use validated dimensions. The last tier is always the fallback.':
      '媒体档位使用已校验的维度，最后一个档位始终作为兜底。',
    'No condition value required': '无需填写条件值',
    'Per image': '张',
    'Per output': '个输出',
    'Per second': '每秒',
    'Per video': '个',
    'Per-second price': '每秒价格',
    'per-second price × duration × output count': '每秒价格 × 时长 × 输出数量',
    'Price preview shows the configured rate. Actual charges use the validated output count and video duration.':
      '价格预览展示配置的费率；实际费用会使用已校验的输出数量和视频时长计算。',
    'Price preview': '价格预览',
    Quality: '质量',
    'Resolution tier': '分辨率档位',
    second: '秒',
    'Settlement delta': '结算差额',
    'Tier condition': '档位条件',
    'Tiers are checked from top to bottom. The first matching tier sets the price.':
      '系统从上到下检查档位，并使用第一个满足条件的档位定价。',
    'The last tier applies when no earlier condition matches, so it does not need a condition value.':
      '当前面的条件都未命中时使用最后一个档位，因此兜底档位不需要填写条件值。',
    'Unit price': '单价',
    Units: '数量',
    'unit price × output count': '单价 × 输出数量',
    'Use quality or image size for images, and resolution for videos. Enter values exactly as supported by the upstream model.':
      '图片可按质量或尺寸区分，视频可按分辨率区分。条件值应与上游模型支持的值完全一致。',
    'Video fixed fee plus duration': '视频固定费用加时长费用',
    'Video price by resolution': '按分辨率计算视频价格',
    'Video resolution tier': '视频分辨率档位',
  },
  'zh-TW': {
    'Upload status': '上傳狀態',
    Uploaded: '已上傳',
    'Human verification is loading': '人機驗證載入中',
    'Human verification is taking longer than expected': '人機驗證載入時間較長',
    'If it does not appear for a long time, refresh the page and try again.':
      '如果長時間沒有出現，請重新整理頁面後再試。',
    'Please wait a moment...': '請稍候...',
    'Proxy Video Content': '代理影片內容',
    'Redirect to the HTTPS URL returned by upstream task details':
      '重新導向到上游任務詳情傳回的 HTTPS 位址',
    'Stream video content through this server for reliable preview and download':
      '由本伺服器轉發影片內容，以獲得穩定的預覽和下載體驗',
    'HTTP Redirect': 'HTTP 重新導向',
    'Require upstream content requests to return a redirect; non-redirect responses fail without proxy fallback':
      '要求上游內容請求傳回重新導向；非重新導向回應直接報錯，不回退代理',
    'Reverse Proxy': '反向代理',
    'Stream video content through this server': '由本伺服器轉發影片內容',
    'Video Content Delivery': '影片內容交付方式',
    '(fixed fee + per-second price × duration) × output count':
      '（固定費用 + 每秒價格 × 時長）× 輸出數量',
    Actual: '實際',
    'Billing method': '計費方式',
    'Choose image quality, image size tier, or exact image size for images; choose video resolution tier for videos. Enter values exactly as supported by the upstream model.':
      '圖片可選擇圖片品質、圖片尺寸級別或具體圖片尺寸；影片請選擇影片解析度級別。條件值應與上游模型支援的值完全一致。',
    'Condition value': '條件值',
    Each: '每個',
    'Enter a provider-supported image quality value, for example high':
      '填寫上游支援的圖片品質值，例如 high',
    'Enter a provider-supported quality value, for example high':
      '填寫上游支援的品質值，例如 high',
    Estimated: '預估',
    'Fixed fee': '固定費用',
    'Fixed fee + per second': '固定費用 + 每秒',
    'Fixed fee plus per second': '固定費用加每秒費用',
    'Generated expression': '產生的運算式',
    'Frimodel mapping: 1024x1024 is 1K; 1536x1024, 1024x1536, auto, and missing size are 2K. Larger custom sizes are classified by their longest edge.':
      'Frimodel 映射：1024x1024 為 1K；1536x1024、1024x1536、auto 和未填寫尺寸均為 2K。更大的自訂尺寸依最長邊分級。',
    'How media pricing works': '媒體價格如何計算',
    'Image price by size': '依尺寸計算圖片價格',
    'Image quality': '圖片品質',
    'Image size': '圖片尺寸',
    'Image size tier': '圖片尺寸級別',
    'Media billing dimensions': '媒體計費維度',
    'Media editor': '媒體編輯器',
    'Media generation': '媒體生成',
    'Media price': '媒體價格',
    'Media tiers use validated dimensions. The last tier is always the fallback.':
      '媒體級別使用已驗證的維度，最後一個級別始終作為後備。',
    'No condition value required': '無需填寫條件值',
    'Per image': '張',
    'Per output': '個輸出',
    'Per second': '每秒',
    'Per video': '個',
    'Per-second price': '每秒價格',
    'per-second price × duration × output count': '每秒價格 × 時長 × 輸出數量',
    'Price preview shows the configured rate. Actual charges use the validated output count and video duration.':
      '價格預覽顯示設定的費率；實際費用會使用已驗證的輸出數量和影片時長計算。',
    'Price preview': '價格預覽',
    Quality: '品質',
    'Resolution tier': '解析度級別',
    second: '秒',
    'Settlement delta': '結算差額',
    'Tier condition': '級別條件',
    'Tiers are checked from top to bottom. The first matching tier sets the price.':
      '系統從上到下檢查級別，並使用第一個符合條件的級別定價。',
    'The last tier applies when no earlier condition matches, so it does not need a condition value.':
      '當前面的條件都未符合時使用最後一個級別，因此後備級別不需要填寫條件值。',
    'Unit price': '單價',
    Units: '數量',
    'unit price × output count': '單價 × 輸出數量',
    'Use quality or image size for images, and resolution for videos. Enter values exactly as supported by the upstream model.':
      '圖片可依品質或尺寸區分，影片可依解析度區分。條件值應與上游模型支援的值完全一致。',
    'Video fixed fee plus duration': '影片固定費用加時長費用',
    'Video price by resolution': '依解析度計算影片價格',
    'Video resolution tier': '影片解析度級別',
  },
  fr: {
    'Upload status': 'État du téléversement',
    Uploaded: 'Téléversé',
    'Human verification is loading':
      'La vérification humaine est en cours de chargement',
    'Human verification is taking longer than expected':
      'La vérification humaine prend plus de temps que prévu',
    'If it does not appear for a long time, refresh the page and try again.':
      'Si elle ne s’affiche pas après un long moment, actualisez la page et réessayez.',
    'Please wait a moment...': 'Veuillez patienter...',
    'Proxy Video Content': 'Proxy du contenu vidéo',
    'Redirect to the HTTPS URL returned by upstream task details':
      'Rediriger vers l’URL HTTPS renvoyée par les détails de la tâche en amont',
    'Stream video content through this server for reliable preview and download':
      'Transmettre la vidéo via ce serveur pour fiabiliser l’aperçu et le téléchargement',
    'HTTP Redirect': 'Redirection HTTP',
    'Require upstream content requests to return a redirect; non-redirect responses fail without proxy fallback':
      'Exiger une redirection du service en amont ; sinon, échouer sans repli vers le proxy',
    'Reverse Proxy': 'Proxy inverse',
    'Stream video content through this server':
      'Diffuser le contenu vidéo via ce serveur',
    'Video Content Delivery': 'Livraison du contenu vidéo',
    '(fixed fee + per-second price × duration) × output count':
      '(frais fixes + prix par seconde × durée) × nombre de sorties',
    Actual: 'Réel',
    'Billing method': 'Mode de facturation',
    'Choose image quality, image size tier, or exact image size for images; choose video resolution tier for videos. Enter values exactly as supported by the upstream model.':
      'Pour les images, choisissez la qualité, le palier de taille ou la taille exacte ; pour les vidéos, choisissez le palier de résolution vidéo. Saisissez exactement les valeurs prises en charge par le modèle en amont.',
    'Condition value': 'Valeur de condition',
    Each: 'Par unité',
    'Enter a provider-supported image quality value, for example high':
      "Saisissez une qualité d'image prise en charge par le fournisseur, par exemple high",
    'Enter a provider-supported quality value, for example high':
      'Saisissez une qualité prise en charge par le fournisseur, par exemple high',
    Estimated: 'Estimé',
    'Fixed fee': 'Frais fixes',
    'Fixed fee + per second': 'Frais fixes + tarif par seconde',
    'Fixed fee plus per second': 'Frais fixes plus tarif par seconde',
    'Generated expression': 'Expression générée',
    'Frimodel mapping: 1024x1024 is 1K; 1536x1024, 1024x1536, auto, and missing size are 2K. Larger custom sizes are classified by their longest edge.':
      'Correspondance Frimodel : 1024x1024 vaut 1K ; 1536x1024, 1024x1536, auto et une taille absente valent 2K. Les tailles personnalisées supérieures sont classées selon leur côté le plus long.',
    'How media pricing works': 'Fonctionnement du tarif multimédia',
    'Image price by size': "Prix de l'image selon la taille",
    'Image quality': "Qualité de l'image",
    'Image size': "Taille de l'image",
    'Image size tier': "Palier de taille d'image",
    'Media billing dimensions': 'Dimensions de facturation multimédia',
    'Media editor': 'Éditeur multimédia',
    'Media generation': 'Génération multimédia',
    'Media price': 'Prix multimédia',
    'Media tiers use validated dimensions. The last tier is always the fallback.':
      'Les paliers multimédias utilisent des dimensions validées. Le dernier palier sert toujours de repli.',
    'No condition value required': 'Aucune valeur de condition requise',
    'Per image': 'image',
    'Per output': 'sortie',
    'Per second': 'Par seconde',
    'Per video': 'vidéo',
    'Per-second price': 'Prix par seconde',
    'per-second price × duration × output count':
      'prix par seconde × durée × nombre de sorties',
    'Price preview shows the configured rate. Actual charges use the validated output count and video duration.':
      'L’aperçu affiche le tarif configuré. Le coût réel utilise le nombre de sorties et la durée vidéo validés.',
    'Price preview': 'Aperçu des prix',
    Quality: 'Qualité',
    'Resolution tier': 'Palier de résolution',
    second: 'seconde',
    'Settlement delta': 'Écart de règlement',
    'Tier condition': 'Condition du palier',
    'Tiers are checked from top to bottom. The first matching tier sets the price.':
      'Les paliers sont vérifiés de haut en bas. Le premier palier correspondant fixe le prix.',
    'The last tier applies when no earlier condition matches, so it does not need a condition value.':
      'Le dernier palier s’applique si aucun précédent ne correspond ; il ne nécessite donc aucune valeur de condition.',
    'Unit price': 'Prix unitaire',
    Units: 'Unités',
    'unit price × output count': 'prix unitaire × nombre de sorties',
    'Use quality or image size for images, and resolution for videos. Enter values exactly as supported by the upstream model.':
      'Utilisez la qualité ou la taille pour les images, et la résolution pour les vidéos. Saisissez exactement les valeurs prises en charge par le modèle en amont.',
    'Video fixed fee plus duration': 'Frais vidéo fixes plus durée',
    'Video price by resolution': 'Prix vidéo selon la résolution',
    'Video resolution tier': 'Palier de résolution vidéo',
  },
  ja: {
    'Upload status': 'アップロード状態',
    Uploaded: 'アップロード済み',
    'Human verification is loading': '人間による確認を読み込み中です',
    'Human verification is taking longer than expected':
      '人間による確認の読み込みに時間がかかっています',
    'If it does not appear for a long time, refresh the page and try again.':
      '長時間表示されない場合は、ページを再読み込みしてもう一度お試しください。',
    'Please wait a moment...': 'しばらくお待ちください...',
    'Proxy Video Content': '動画コンテンツをプロキシ',
    'Redirect to the HTTPS URL returned by upstream task details':
      '上流のタスク詳細が返す HTTPS URL にリダイレクトします',
    'Stream video content through this server for reliable preview and download':
      '安定したプレビューとダウンロードのため、このサーバー経由で動画を配信します',
    'HTTP Redirect': 'HTTP リダイレクト',
    'Require upstream content requests to return a redirect; non-redirect responses fail without proxy fallback':
      '上流のコンテンツ要求にリダイレクト応答を必須とし、それ以外はプロキシへフォールバックせず失敗させます',
    'Reverse Proxy': 'リバースプロキシ',
    'Stream video content through this server':
      'このサーバー経由で動画コンテンツを配信します',
    'Video Content Delivery': '動画コンテンツの配信方式',
    '(fixed fee + per-second price × duration) × output count':
      '（固定料金 + 秒単価 × 時間）× 出力数',
    Actual: '実績',
    'Billing method': '課金方式',
    'Choose image quality, image size tier, or exact image size for images; choose video resolution tier for videos. Enter values exactly as supported by the upstream model.':
      '画像には画像品質、画像サイズ階層、または正確な画像サイズを選択し、動画には動画解像度階層を選択します。条件値は上流モデルが対応する値と完全に一致させてください。',
    'Condition value': '条件値',
    Each: '1件ごと',
    'Enter a provider-supported image quality value, for example high':
      'プロバイダー対応の画像品質値を入力（例: high）',
    'Enter a provider-supported quality value, for example high':
      'プロバイダー対応の品質値を入力（例: high）',
    Estimated: '見積もり',
    'Fixed fee': '固定料金',
    'Fixed fee + per second': '固定料金 + 秒単価',
    'Fixed fee plus per second': '固定料金と秒単価',
    'Generated expression': '生成された式',
    'Frimodel mapping: 1024x1024 is 1K; 1536x1024, 1024x1536, auto, and missing size are 2K. Larger custom sizes are classified by their longest edge.':
      'Frimodel の分類: 1024x1024 は 1K、1536x1024、1024x1536、auto、サイズ未指定は 2K です。より大きなカスタムサイズは長辺で分類されます。',
    'How media pricing works': 'メディア料金の計算方法',
    'Image price by size': 'サイズ別画像料金',
    'Image quality': '画像品質',
    'Image size': '画像サイズ',
    'Image size tier': '画像サイズ階層',
    'Media billing dimensions': 'メディア課金ディメンション',
    'Media editor': 'メディアエディター',
    'Media generation': 'メディア生成',
    'Media price': 'メディア料金',
    'Media tiers use validated dimensions. The last tier is always the fallback.':
      'メディア階層には検証済みのディメンションを使用します。最後の階層は常にフォールバックです。',
    'No condition value required': '条件値の入力は不要です',
    'Per image': '画像',
    'Per output': '出力',
    'Per second': '秒ごと',
    'Per video': '動画',
    'Per-second price': '秒単価',
    'per-second price × duration × output count': '秒単価 × 時間 × 出力数',
    'Price preview shows the configured rate. Actual charges use the validated output count and video duration.':
      '料金プレビューには設定レートが表示されます。実際の料金は検証済みの出力数と動画時間で計算されます。',
    'Price preview': '料金プレビュー',
    Quality: '品質',
    'Resolution tier': '解像度階層',
    second: '秒',
    'Settlement delta': '精算差額',
    'Tier condition': '階層条件',
    'Tiers are checked from top to bottom. The first matching tier sets the price.':
      '階層は上から順に判定され、最初に一致した階層の料金が適用されます。',
    'The last tier applies when no earlier condition matches, so it does not need a condition value.':
      '最後の階層はそれ以前の条件に一致しない場合に適用されるため、条件値は不要です。',
    'Unit price': '単価',
    Units: '数量',
    'unit price × output count': '単価 × 出力数',
    'Use quality or image size for images, and resolution for videos. Enter values exactly as supported by the upstream model.':
      '画像には品質またはサイズ、動画には解像度を使用します。条件値は上流モデルが対応する値と完全に一致させてください。',
    'Video fixed fee plus duration': '動画の固定料金と時間料金',
    'Video price by resolution': '解像度別動画料金',
    'Video resolution tier': '動画解像度階層',
  },
  ru: {
    'Upload status': 'Статус загрузки',
    Uploaded: 'Загружено',
    'Human verification is loading': 'Проверка на человека загружается',
    'Human verification is taking longer than expected':
      'Проверка на человека загружается дольше, чем ожидалось',
    'If it does not appear for a long time, refresh the page and try again.':
      'Если проверка долго не появляется, обновите страницу и попробуйте снова.',
    'Please wait a moment...': 'Пожалуйста, подождите...',
    'Proxy Video Content': 'Проксировать видео',
    'Redirect to the HTTPS URL returned by upstream task details':
      'Перенаправлять на HTTPS-адрес из сведений о задаче вышестоящего сервиса',
    'Stream video content through this server for reliable preview and download':
      'Передавать видео через этот сервер для стабильного просмотра и скачивания',
    'HTTP Redirect': 'HTTP-перенаправление',
    'Require upstream content requests to return a redirect; non-redirect responses fail without proxy fallback':
      'Требовать перенаправление от вышестоящего сервиса; при другом ответе завершать запрос с ошибкой без перехода на прокси',
    'Reverse Proxy': 'Обратный прокси',
    'Stream video content through this server':
      'Передавать видеоконтент через этот сервер',
    'Video Content Delivery': 'Доставка видеоконтента',
    '(fixed fee + per-second price × duration) × output count':
      '(фиксированная плата + цена за секунду × длительность) × число результатов',
    Actual: 'Фактически',
    'Billing method': 'Способ тарификации',
    'Choose image quality, image size tier, or exact image size for images; choose video resolution tier for videos. Enter values exactly as supported by the upstream model.':
      'Для изображений выберите качество, уровень размера или точный размер изображения; для видео выберите уровень разрешения видео. Значения должны точно соответствовать поддерживаемым вышестоящей моделью.',
    'Condition value': 'Значение условия',
    Each: 'За единицу',
    'Enter a provider-supported image quality value, for example high':
      'Введите значение качества изображения, поддерживаемое поставщиком, например high',
    'Enter a provider-supported quality value, for example high':
      'Введите значение качества, поддерживаемое поставщиком, например high',
    Estimated: 'Оценка',
    'Fixed fee': 'Фиксированная плата',
    'Fixed fee + per second': 'Фиксированная плата + цена за секунду',
    'Fixed fee plus per second': 'Фиксированная плата плюс цена за секунду',
    'Generated expression': 'Сформированное выражение',
    'Frimodel mapping: 1024x1024 is 1K; 1536x1024, 1024x1536, auto, and missing size are 2K. Larger custom sizes are classified by their longest edge.':
      'Сопоставление Frimodel: 1024x1024 — 1K; 1536x1024, 1024x1536, auto и отсутствие размера — 2K. Более крупные пользовательские размеры классифицируются по длинной стороне.',
    'How media pricing works': 'Как рассчитывается цена медиа',
    'Image price by size': 'Цена изображения по размеру',
    'Image quality': 'Качество изображения',
    'Image size': 'Размер изображения',
    'Image size tier': 'Уровень размера изображения',
    'Media billing dimensions': 'Параметры тарификации медиа',
    'Media editor': 'Редактор медиа',
    'Media generation': 'Генерация медиа',
    'Media price': 'Цена медиа',
    'Media tiers use validated dimensions. The last tier is always the fallback.':
      'Уровни медиа используют проверенные параметры. Последний уровень всегда является резервным.',
    'No condition value required': 'Значение условия не требуется',
    'Per image': 'изображение',
    'Per output': 'результат',
    'Per second': 'За секунду',
    'Per video': 'видео',
    'Per-second price': 'Цена за секунду',
    'per-second price × duration × output count':
      'цена за секунду × длительность × число результатов',
    'Price preview shows the configured rate. Actual charges use the validated output count and video duration.':
      'Предпросмотр показывает настроенный тариф. Фактическая плата рассчитывается по проверенному числу результатов и длительности видео.',
    'Price preview': 'Предпросмотр цены',
    Quality: 'Качество',
    'Resolution tier': 'Уровень разрешения',
    second: 'секунда',
    'Settlement delta': 'Разница расчёта',
    'Tier condition': 'Условие уровня',
    'Tiers are checked from top to bottom. The first matching tier sets the price.':
      'Уровни проверяются сверху вниз. Цена определяется первым подходящим уровнем.',
    'The last tier applies when no earlier condition matches, so it does not need a condition value.':
      'Последний уровень применяется, если предыдущие условия не подошли, поэтому значение условия для него не требуется.',
    'Unit price': 'Цена за единицу',
    Units: 'Единицы',
    'unit price × output count': 'цена за единицу × число результатов',
    'Use quality or image size for images, and resolution for videos. Enter values exactly as supported by the upstream model.':
      'Для изображений используйте качество или размер, для видео — разрешение. Значения должны точно соответствовать поддерживаемым вышестоящей моделью.',
    'Video fixed fee plus duration':
      'Фиксированная плата за видео плюс длительность',
    'Video price by resolution': 'Цена видео по разрешению',
    'Video resolution tier': 'Уровень разрешения видео',
  },
  vi: {
    'Upload status': 'Trạng thái tải lên',
    Uploaded: 'Đã tải lên',
    'Human verification is loading': 'Đang tải bước xác minh người dùng',
    'Human verification is taking longer than expected':
      'Quá trình xác minh người dùng đang mất nhiều thời gian hơn dự kiến',
    'If it does not appear for a long time, refresh the page and try again.':
      'Nếu bước xác minh không xuất hiện sau một thời gian dài, hãy tải lại trang và thử lại.',
    'Please wait a moment...': 'Vui lòng chờ một lát...',
    'Proxy Video Content': 'Proxy nội dung video',
    'Redirect to the HTTPS URL returned by upstream task details':
      'Chuyển hướng đến URL HTTPS do chi tiết tác vụ thượng nguồn trả về',
    'Stream video content through this server for reliable preview and download':
      'Truyền video qua máy chủ này để xem trước và tải xuống ổn định',
    'HTTP Redirect': 'Chuyển hướng HTTP',
    'Require upstream content requests to return a redirect; non-redirect responses fail without proxy fallback':
      'Yêu cầu thượng nguồn trả về chuyển hướng; nếu không, báo lỗi mà không chuyển sang proxy',
    'Reverse Proxy': 'Proxy ngược',
    'Stream video content through this server':
      'Truyền nội dung video qua máy chủ này',
    'Video Content Delivery': 'Phương thức phân phối nội dung video',
    '(fixed fee + per-second price × duration) × output count':
      '(phí cố định + giá mỗi giây × thời lượng) × số đầu ra',
    Actual: 'Thực tế',
    'Billing method': 'Phương thức tính phí',
    'Choose image quality, image size tier, or exact image size for images; choose video resolution tier for videos. Enter values exactly as supported by the upstream model.':
      'Với ảnh, hãy chọn chất lượng ảnh, bậc kích thước ảnh hoặc kích thước ảnh chính xác; với video, hãy chọn bậc độ phân giải video. Nhập chính xác các giá trị mà mô hình thượng nguồn hỗ trợ.',
    'Condition value': 'Giá trị điều kiện',
    Each: 'Mỗi đơn vị',
    'Enter a provider-supported image quality value, for example high':
      'Nhập giá trị chất lượng ảnh được nhà cung cấp hỗ trợ, ví dụ high',
    'Enter a provider-supported quality value, for example high':
      'Nhập giá trị chất lượng được nhà cung cấp hỗ trợ, ví dụ high',
    Estimated: 'Ước tính',
    'Fixed fee': 'Phí cố định',
    'Fixed fee + per second': 'Phí cố định + phí mỗi giây',
    'Fixed fee plus per second': 'Phí cố định cộng phí mỗi giây',
    'Generated expression': 'Biểu thức đã tạo',
    'Frimodel mapping: 1024x1024 is 1K; 1536x1024, 1024x1536, auto, and missing size are 2K. Larger custom sizes are classified by their longest edge.':
      'Ánh xạ Frimodel: 1024x1024 là 1K; 1536x1024, 1024x1536, auto và không nhập kích thước là 2K. Kích thước tùy chỉnh lớn hơn được phân loại theo cạnh dài nhất.',
    'How media pricing works': 'Cách tính giá nội dung đa phương tiện',
    'Image price by size': 'Giá ảnh theo kích thước',
    'Image quality': 'Chất lượng ảnh',
    'Image size': 'Kích thước ảnh',
    'Image size tier': 'Bậc kích thước ảnh',
    'Media billing dimensions': 'Thông số tính phí nội dung đa phương tiện',
    'Media editor': 'Trình chỉnh sửa nội dung đa phương tiện',
    'Media generation': 'Tạo nội dung đa phương tiện',
    'Media price': 'Giá nội dung đa phương tiện',
    'Media tiers use validated dimensions. The last tier is always the fallback.':
      'Các bậc nội dung đa phương tiện dùng thông số đã xác thực. Bậc cuối luôn là phương án dự phòng.',
    'No condition value required': 'Không cần nhập giá trị điều kiện',
    'Per image': 'ảnh',
    'Per output': 'đầu ra',
    'Per second': 'Mỗi giây',
    'Per video': 'video',
    'Per-second price': 'Giá mỗi giây',
    'per-second price × duration × output count':
      'giá mỗi giây × thời lượng × số đầu ra',
    'Price preview shows the configured rate. Actual charges use the validated output count and video duration.':
      'Phần xem trước hiển thị mức giá đã cấu hình. Chi phí thực tế dùng số đầu ra và thời lượng video đã xác thực.',
    'Price preview': 'Xem trước giá',
    Quality: 'Chất lượng',
    'Resolution tier': 'Bậc độ phân giải',
    second: 'giây',
    'Settlement delta': 'Chênh lệch quyết toán',
    'Tier condition': 'Điều kiện bậc',
    'Tiers are checked from top to bottom. The first matching tier sets the price.':
      'Các bậc được kiểm tra từ trên xuống. Bậc khớp đầu tiên sẽ xác định giá.',
    'The last tier applies when no earlier condition matches, so it does not need a condition value.':
      'Bậc cuối được áp dụng khi không có điều kiện trước đó khớp, vì vậy không cần giá trị điều kiện.',
    'Unit price': 'Đơn giá',
    Units: 'Số lượng',
    'unit price × output count': 'đơn giá × số đầu ra',
    'Use quality or image size for images, and resolution for videos. Enter values exactly as supported by the upstream model.':
      'Dùng chất lượng hoặc kích thước cho ảnh và độ phân giải cho video. Nhập chính xác các giá trị mà mô hình thượng nguồn hỗ trợ.',
    'Video fixed fee plus duration': 'Phí video cố định cộng phí thời lượng',
    'Video price by resolution': 'Giá video theo độ phân giải',
    'Video resolution tier': 'Bậc độ phân giải video',
  },
}

Object.assign(newKeys.en, {
  'Async Image Tasks': 'Async Image Tasks',
  'Object Storage': 'Object Storage',
  'Object retention (seconds)': 'Object retention (seconds)',
  'Presigned URL lifetime (seconds)': 'Presigned URL lifetime (seconds)',
  'Archive attempt timeout (seconds)': 'Archive attempt timeout (seconds)',
  'Maximum archive attempts': 'Maximum archive attempts',
  'Maximum retry window (seconds)': 'Maximum retry window (seconds)',
  'Cleanup interval (seconds)': 'Cleanup interval (seconds)',
  'Region is required': 'Region is required',
  'Bucket is required': 'Bucket is required',
  'Bucket cannot contain spaces': 'Bucket cannot contain spaces',
  'Access Key is required': 'Access Key is required',
  'Retry window must not be shorter than archive timeout':
    'Retry window must not be shorter than archive timeout',
  'Object storage settings saved': 'Object storage settings saved',
  'Failed to save object storage settings':
    'Failed to save object storage settings',
  'Save settings': 'Save settings',
  'Secret storage notice': 'Secret storage notice',
  'The S3 Secret Access Key is stored as plaintext in the database. Anyone with database or backup access can read it.':
    'The S3 Secret Access Key is stored as plaintext in the database. Anyone with database or backup access can read it.',
  'S3 connection': 'S3 connection',
  'Objects remain private and are delivered with temporary signed URLs.':
    'Objects remain private and are delivered with temporary signed URLs.',
  'Secret configured': 'Secret configured',
  'Secret not configured': 'Secret not configured',
  'Endpoint URL': 'Endpoint URL',
  'Leave blank to use the standard AWS endpoint.':
    'Leave blank to use the standard AWS endpoint.',
  'Secret Access Key': 'Secret Access Key',
  'Leave blank to keep the existing secret':
    'Leave blank to keep the existing secret',
  'Request parameters were not retained for this historical task':
    'Request parameters were not retained for this historical task',
  'Retry selected uploads': 'Retry selected uploads',
  'Retry all failed image uploads': 'Retry all failed image uploads',
  'Retry selected': 'Retry selected',
  'Retry all failed': 'Retry all failed',
  'Retry all failed uploads?': 'Retry all failed uploads?',
  'Only persistent staged files are uploaded again. Image generation and billing are not repeated.':
    'Only persistent staged files are uploaded again. Image generation and billing are not repeated.',
  'Start retry': 'Start retry',
  'Generation status': 'Generation status',
  'Output availability': 'Output availability',
  'Billing status': 'Billing status',
  Objects: 'Objects',
  Attempts: 'Attempts',
  'Staging integrity': 'Staging integrity',
  'Created at': 'Created at',
  'Select retryable tasks': 'Select retryable tasks',
  'Select task {{taskId}}': 'Select task {{taskId}}',
  'No async image tasks found': 'No async image tasks found',
  '{{count}} tasks': '{{count}} tasks',
  'Accepted {{accepted}} tasks; skipped {{skipped}} tasks':
    'Accepted {{accepted}} tasks; skipped {{skipped}} tasks',
  'Failed to retry image uploads': 'Failed to retry image uploads',
  'Bulk retry operation started: {{operationId}}':
    'Bulk retry operation started: {{operationId}}',
  'Failed to start bulk retry': 'Failed to start bulk retry',
  queued: 'Queued',
  archiving: 'Archiving',
  available: 'Available',
  reserved: 'Reserved',
  settled: 'Settled',
  refunded: 'Refunded',
  not_staged: 'Not staged',
})

Object.assign(newKeys.zh, {
  'Async Image Tasks': '异步图片任务',
  'Object Storage': '对象存储',
  'Object retention (seconds)': '对象保留时间（秒）',
  'Presigned URL lifetime (seconds)': '预签名 URL 有效期（秒）',
  'Archive attempt timeout (seconds)': '单次归档超时（秒）',
  'Maximum archive attempts': '最大归档尝试次数',
  'Maximum retry window (seconds)': '最长重试窗口（秒）',
  'Cleanup interval (seconds)': '清理间隔（秒）',
  'Region is required': 'Region 不能为空',
  'Bucket is required': 'Bucket 不能为空',
  'Bucket cannot contain spaces': 'Bucket 不能包含空格',
  'Access Key is required': 'Access Key 不能为空',
  'Retry window must not be shorter than archive timeout':
    '重试窗口不能短于单次归档超时',
  'Object storage settings saved': '对象存储设置已保存',
  'Failed to save object storage settings': '保存对象存储设置失败',
  'Save settings': '保存设置',
  'Secret storage notice': 'Secret 存储提示',
  'The S3 Secret Access Key is stored as plaintext in the database. Anyone with database or backup access can read it.':
    'S3 Secret Access Key 会以明文存入数据库，拥有数据库或备份读取权限的人可以查看。',
  'S3 connection': 'S3 连接',
  'Objects remain private and are delivered with temporary signed URLs.':
    '对象保持私有，仅通过临时签名 URL 交付。',
  'Secret configured': 'Secret 已配置',
  'Secret not configured': 'Secret 未配置',
  'Endpoint URL': '端点 URL',
  'Leave blank to use the standard AWS endpoint.': '留空则使用 AWS 标准端点。',
  'Secret Access Key': 'Secret 访问密钥',
  'Leave blank to keep the existing secret': '留空以保留现有 Secret',
  'Request parameters were not retained for this historical task':
    '历史任务未保留请求参数',
  'Retry selected uploads': '重新上传选中项',
  'Retry all failed image uploads': '重新上传全部失败的图片文件',
  'Retry selected': '重新上传选中项',
  'Retry all failed': '重新上传全部失败项',
  'Retry all failed uploads?': '重新上传全部失败文件？',
  'Only persistent staged files are uploaded again. Image generation and billing are not repeated.':
    '只会重新上传持久暂存文件，不会重新生成图片或重复扣费。',
  'Start retry': '开始重试',
  'Generation status': '生成状态',
  'Output availability': '输出可用性',
  'Billing status': '计费状态',
  Objects: '对象',
  Attempts: '尝试次数',
  'Staging integrity': '暂存完整性',
  'Created at': '创建时间',
  'Select retryable tasks': '选择可重试任务',
  'Select task {{taskId}}': '选择任务 {{taskId}}',
  'No async image tasks found': '未找到异步图片任务',
  '{{count}} tasks': '{{count}} 个任务',
  'Accepted {{accepted}} tasks; skipped {{skipped}} tasks':
    '已接受 {{accepted}} 个任务，跳过 {{skipped}} 个任务',
  'Failed to retry image uploads': '重新上传图片失败',
  'Bulk retry operation started: {{operationId}}':
    '批量重试已启动：{{operationId}}',
  'Failed to start bulk retry': '启动批量重试失败',
  queued: '排队中',
  archiving: '归档中',
  available: '可用',
  reserved: '已预扣',
  settled: '已结算',
  refunded: '已退款',
  not_staged: '未暂存',
})

Object.assign(newKeys['zh-TW'], {
  'Async Image Tasks': '非同步圖片任務',
  'Object Storage': '物件儲存',
  'Object retention (seconds)': '物件保留時間（秒）',
  'Presigned URL lifetime (seconds)': '預簽名 URL 有效期（秒）',
  'Archive attempt timeout (seconds)': '單次封存逾時（秒）',
  'Maximum archive attempts': '最大封存嘗試次數',
  'Maximum retry window (seconds)': '最長重試視窗（秒）',
  'Cleanup interval (seconds)': '清理間隔（秒）',
  'Region is required': 'Region 為必填',
  'Bucket is required': 'Bucket 為必填',
  'Bucket cannot contain spaces': 'Bucket 不可包含空格',
  'Access Key is required': 'Access Key 為必填',
  'Retry window must not be shorter than archive timeout':
    '重試視窗不可短於單次封存逾時',
  'Object storage settings saved': '物件儲存設定已儲存',
  'Failed to save object storage settings': '儲存物件儲存設定失敗',
  'Save settings': '儲存設定',
  'Secret storage notice': 'Secret 儲存提示',
  'The S3 Secret Access Key is stored as plaintext in the database. Anyone with database or backup access can read it.':
    'S3 Secret Access Key 會以明文存入資料庫，擁有資料庫或備份讀取權限的人可查看。',
  'S3 connection': 'S3 連線',
  'Objects remain private and are delivered with temporary signed URLs.':
    '物件維持私有，僅透過臨時簽名 URL 交付。',
  'Secret configured': 'Secret 已設定',
  'Secret not configured': 'Secret 未設定',
  'Endpoint URL': 'Endpoint URL',
  'Leave blank to use the standard AWS endpoint.': '留空以使用 AWS 標準端點。',
  'Secret Access Key': 'Secret Access Key',
  'Leave blank to keep the existing secret': '留空以保留現有 Secret',
  'Request parameters were not retained for this historical task':
    '歷史任務未保留請求參數',
  'Retry selected uploads': '重新上傳所選項目',
  'Retry all failed image uploads': '重新上傳全部失敗的圖片檔案',
  'Retry selected': '重新上傳所選項目',
  'Retry all failed': '重新上傳全部失敗項目',
  'Retry all failed uploads?': '重新上傳全部失敗檔案？',
  'Only persistent staged files are uploaded again. Image generation and billing are not repeated.':
    '只會重新上傳持久暫存檔案，不會重新生成圖片或重複計費。',
  'Start retry': '開始重試',
  'Generation status': '生成狀態',
  'Output availability': '輸出可用性',
  'Billing status': '計費狀態',
  Objects: '物件',
  Attempts: '嘗試次數',
  'Staging integrity': '暫存完整性',
  'Created at': '建立時間',
  'Select retryable tasks': '選擇可重試任務',
  'Select task {{taskId}}': '選擇任務 {{taskId}}',
  'No async image tasks found': '找不到非同步圖片任務',
  '{{count}} tasks': '{{count}} 個任務',
  'Accepted {{accepted}} tasks; skipped {{skipped}} tasks':
    '已接受 {{accepted}} 個任務，略過 {{skipped}} 個任務',
  'Failed to retry image uploads': '重新上傳圖片失敗',
  'Bulk retry operation started: {{operationId}}':
    '批次重試已啟動：{{operationId}}',
  'Failed to start bulk retry': '啟動批次重試失敗',
  queued: '排隊中',
  archiving: '封存中',
  available: '可用',
  reserved: '已預扣',
  settled: '已結算',
  refunded: '已退款',
  not_staged: '未暫存',
})

Object.assign(newKeys.fr, {
  'Async Image Tasks': 'Tâches d’images asynchrones',
  'Object Storage': 'Stockage d’objets',
  'Object retention (seconds)': 'Conservation des objets (secondes)',
  'Presigned URL lifetime (seconds)': 'Durée des URL signées (secondes)',
  'Archive attempt timeout (seconds)': 'Délai d’archivage (secondes)',
  'Maximum archive attempts': 'Nombre maximal de tentatives',
  'Maximum retry window (seconds)': 'Fenêtre maximale de reprise (secondes)',
  'Cleanup interval (seconds)': 'Intervalle de nettoyage (secondes)',
  'Region is required': 'La région est obligatoire',
  'Bucket is required': 'Le bucket est obligatoire',
  'Bucket cannot contain spaces': 'Le bucket ne peut pas contenir d’espaces',
  'Access Key is required': 'La clé d’accès est obligatoire',
  'Retry window must not be shorter than archive timeout':
    'La fenêtre de reprise doit dépasser le délai d’archivage',
  'Object storage settings saved': 'Paramètres de stockage enregistrés',
  'Failed to save object storage settings':
    'Échec de l’enregistrement du stockage',
  'Save settings': 'Enregistrer',
  'Secret storage notice': 'Avis sur le stockage du secret',
  'The S3 Secret Access Key is stored as plaintext in the database. Anyone with database or backup access can read it.':
    'La clé secrète S3 est stockée en clair dans la base. Toute personne ayant accès à la base ou aux sauvegardes peut la lire.',
  'S3 connection': 'Connexion S3',
  'Objects remain private and are delivered with temporary signed URLs.':
    'Les objets restent privés et sont fournis via des URL signées temporaires.',
  'Secret configured': 'Secret configuré',
  'Secret not configured': 'Secret non configuré',
  'Endpoint URL': 'URL du point de terminaison',
  'Leave blank to use the standard AWS endpoint.':
    'Laissez vide pour utiliser le point de terminaison AWS standard.',
  'Secret Access Key': 'Clé d’accès secrète',
  'Leave blank to keep the existing secret':
    'Laissez vide pour conserver le secret actuel',
  'Request parameters were not retained for this historical task':
    "Les paramètres de requête n'ont pas été conservés pour cette tâche historique",
  'Retry selected uploads': 'Relancer les envois sélectionnés',
  'Retry all failed image uploads': 'Relancer tous les envois d’images échoués',
  'Retry selected': 'Relancer la sélection',
  'Retry all failed': 'Relancer tous les échecs',
  'Retry all failed uploads?': 'Relancer tous les envois échoués ?',
  'Only persistent staged files are uploaded again. Image generation and billing are not repeated.':
    'Seuls les fichiers persistants sont renvoyés. La génération et la facturation ne sont pas répétées.',
  'Start retry': 'Démarrer la reprise',
  'Generation status': 'État de génération',
  'Output availability': 'Disponibilité du résultat',
  'Billing status': 'État de facturation',
  Objects: 'Objets',
  Attempts: 'Tentatives',
  'Staging integrity': 'Intégrité du stockage temporaire',
  'Created at': 'Créé le',
  'Select retryable tasks': 'Sélectionner les tâches relançables',
  'Select task {{taskId}}': 'Sélectionner la tâche {{taskId}}',
  'No async image tasks found': 'Aucune tâche d’image asynchrone',
  '{{count}} tasks': '{{count}} tâches',
  'Accepted {{accepted}} tasks; skipped {{skipped}} tasks':
    '{{accepted}} tâches acceptées, {{skipped}} ignorées',
  'Failed to retry image uploads': 'Échec de la reprise des envois',
  'Bulk retry operation started: {{operationId}}':
    'Reprise groupée démarrée : {{operationId}}',
  'Failed to start bulk retry': 'Échec du démarrage de la reprise groupée',
  queued: 'En attente',
  archiving: 'Archivage',
  available: 'Disponible',
  reserved: 'Réservé',
  settled: 'Réglé',
  refunded: 'Remboursé',
  not_staged: 'Non stocké',
})

Object.assign(newKeys.ja, {
  'Async Image Tasks': '非同期画像タスク',
  'Object Storage': 'オブジェクトストレージ',
  'Object retention (seconds)': 'オブジェクト保持期間（秒）',
  'Presigned URL lifetime (seconds)': '署名付き URL の有効期間（秒）',
  'Archive attempt timeout (seconds)': 'アーカイブ試行タイムアウト（秒）',
  'Maximum archive attempts': '最大アーカイブ試行回数',
  'Maximum retry window (seconds)': '最大再試行期間（秒）',
  'Cleanup interval (seconds)': 'クリーンアップ間隔（秒）',
  'Region is required': 'リージョンは必須です',
  'Bucket is required': 'バケットは必須です',
  'Bucket cannot contain spaces': 'バケットに空白は使用できません',
  'Access Key is required': 'アクセスキーは必須です',
  'Retry window must not be shorter than archive timeout':
    '再試行期間はアーカイブタイムアウト以上にしてください',
  'Object storage settings saved': 'オブジェクトストレージ設定を保存しました',
  'Failed to save object storage settings': '設定の保存に失敗しました',
  'Save settings': '設定を保存',
  'Secret storage notice': 'シークレット保存に関する注意',
  'The S3 Secret Access Key is stored as plaintext in the database. Anyone with database or backup access can read it.':
    'S3 シークレットアクセスキーはデータベースに平文で保存され、DB またはバックアップの閲覧権限者が読み取れます。',
  'S3 connection': 'S3 接続',
  'Objects remain private and are delivered with temporary signed URLs.':
    'オブジェクトは非公開のまま、一時的な署名付き URL で提供されます。',
  'Secret configured': 'シークレット設定済み',
  'Secret not configured': 'シークレット未設定',
  'Endpoint URL': 'エンドポイント URL',
  'Leave blank to use the standard AWS endpoint.':
    '空欄の場合は標準の AWS エンドポイントを使用します。',
  'Secret Access Key': 'シークレットアクセスキー',
  'Leave blank to keep the existing secret':
    '既存のシークレットを維持する場合は空欄',
  'Request parameters were not retained for this historical task':
    '過去のタスクではリクエストパラメータが保存されていません',
  'Retry selected uploads': '選択したアップロードを再試行',
  'Retry all failed image uploads': '失敗した画像アップロードをすべて再試行',
  'Retry selected': '選択項目を再試行',
  'Retry all failed': '失敗分をすべて再試行',
  'Retry all failed uploads?': '失敗したアップロードをすべて再試行しますか？',
  'Only persistent staged files are uploaded again. Image generation and billing are not repeated.':
    '永続ステージファイルのみ再アップロードします。画像生成と課金は繰り返しません。',
  'Start retry': '再試行を開始',
  'Generation status': '生成ステータス',
  'Output availability': '出力の可用性',
  'Billing status': '課金ステータス',
  Objects: 'オブジェクト',
  Attempts: '試行回数',
  'Staging integrity': 'ステージ整合性',
  'Created at': '作成日時',
  'Select retryable tasks': '再試行可能なタスクを選択',
  'Select task {{taskId}}': 'タスク {{taskId}} を選択',
  'No async image tasks found': '非同期画像タスクがありません',
  '{{count}} tasks': '{{count}} 件のタスク',
  'Accepted {{accepted}} tasks; skipped {{skipped}} tasks':
    '{{accepted}} 件を受理し、{{skipped}} 件をスキップしました',
  'Failed to retry image uploads': '画像アップロードの再試行に失敗しました',
  'Bulk retry operation started: {{operationId}}':
    '一括再試行を開始しました: {{operationId}}',
  'Failed to start bulk retry': '一括再試行を開始できませんでした',
  queued: '待機中',
  archiving: 'アーカイブ中',
  available: '利用可能',
  reserved: '予約済み',
  settled: '精算済み',
  refunded: '返金済み',
  not_staged: '未ステージ',
})

Object.assign(newKeys.ru, {
  'Async Image Tasks': 'Асинхронные задачи изображений',
  'Object Storage': 'Объектное хранилище',
  'Object retention (seconds)': 'Срок хранения объектов (сек.)',
  'Presigned URL lifetime (seconds)': 'Срок действия подписанной URL (сек.)',
  'Archive attempt timeout (seconds)': 'Тайм-аут архивации (сек.)',
  'Maximum archive attempts': 'Максимум попыток архивации',
  'Maximum retry window (seconds)': 'Максимальное окно повторов (сек.)',
  'Cleanup interval (seconds)': 'Интервал очистки (сек.)',
  'Region is required': 'Укажите регион',
  'Bucket is required': 'Укажите бакет',
  'Bucket cannot contain spaces': 'Бакет не может содержать пробелы',
  'Access Key is required': 'Укажите ключ доступа',
  'Retry window must not be shorter than archive timeout':
    'Окно повторов не должно быть короче тайм-аута архивации',
  'Object storage settings saved': 'Настройки хранилища сохранены',
  'Failed to save object storage settings':
    'Не удалось сохранить настройки хранилища',
  'Save settings': 'Сохранить настройки',
  'Secret storage notice': 'Уведомление о хранении секрета',
  'The S3 Secret Access Key is stored as plaintext in the database. Anyone with database or backup access can read it.':
    'Секретный ключ S3 хранится в базе открытым текстом и доступен лицам с доступом к базе или резервным копиям.',
  'S3 connection': 'Подключение S3',
  'Objects remain private and are delivered with temporary signed URLs.':
    'Объекты остаются закрытыми и выдаются по временным подписанным URL.',
  'Secret configured': 'Секрет настроен',
  'Secret not configured': 'Секрет не настроен',
  'Endpoint URL': 'URL конечной точки',
  'Leave blank to use the standard AWS endpoint.':
    'Оставьте пустым для стандартной точки AWS.',
  'Secret Access Key': 'Секретный ключ доступа',
  'Leave blank to keep the existing secret':
    'Оставьте пустым, чтобы сохранить текущий секрет',
  'Request parameters were not retained for this historical task':
    'Параметры запроса не сохранены для этой исторической задачи',
  'Retry selected uploads': 'Повторить выбранные загрузки',
  'Retry all failed image uploads':
    'Повторить все неудачные загрузки изображений',
  'Retry selected': 'Повторить выбранное',
  'Retry all failed': 'Повторить все ошибки',
  'Retry all failed uploads?': 'Повторить все неудачные загрузки?',
  'Only persistent staged files are uploaded again. Image generation and billing are not repeated.':
    'Повторно загружаются только сохранённые временные файлы. Генерация и списание не повторяются.',
  'Start retry': 'Начать повтор',
  'Generation status': 'Статус генерации',
  'Output availability': 'Доступность результата',
  'Billing status': 'Статус оплаты',
  Objects: 'Объекты',
  Attempts: 'Попытки',
  'Staging integrity': 'Целостность временных файлов',
  'Created at': 'Создано',
  'Select retryable tasks': 'Выбрать задачи для повтора',
  'Select task {{taskId}}': 'Выбрать задачу {{taskId}}',
  'No async image tasks found': 'Асинхронные задачи изображений не найдены',
  '{{count}} tasks': 'Задач: {{count}}',
  'Accepted {{accepted}} tasks; skipped {{skipped}} tasks':
    'Принято: {{accepted}}, пропущено: {{skipped}}',
  'Failed to retry image uploads': 'Не удалось повторить загрузку изображений',
  'Bulk retry operation started: {{operationId}}':
    'Массовый повтор запущен: {{operationId}}',
  'Failed to start bulk retry': 'Не удалось запустить массовый повтор',
  queued: 'В очереди',
  archiving: 'Архивация',
  available: 'Доступно',
  reserved: 'Зарезервировано',
  settled: 'Рассчитано',
  refunded: 'Возвращено',
  not_staged: 'Не сохранено',
})

Object.assign(newKeys.vi, {
  'Async Image Tasks': 'Tác vụ ảnh bất đồng bộ',
  'Object Storage': 'Lưu trữ đối tượng',
  'Object retention (seconds)': 'Thời gian lưu đối tượng (giây)',
  'Presigned URL lifetime (seconds)': 'Thời hạn URL ký sẵn (giây)',
  'Archive attempt timeout (seconds)': 'Thời gian chờ lưu trữ (giây)',
  'Maximum archive attempts': 'Số lần lưu trữ tối đa',
  'Maximum retry window (seconds)': 'Khoảng thử lại tối đa (giây)',
  'Cleanup interval (seconds)': 'Chu kỳ dọn dẹp (giây)',
  'Region is required': 'Bắt buộc nhập khu vực',
  'Bucket is required': 'Bắt buộc nhập bucket',
  'Bucket cannot contain spaces': 'Bucket không được chứa khoảng trắng',
  'Access Key is required': 'Bắt buộc nhập Access Key',
  'Retry window must not be shorter than archive timeout':
    'Khoảng thử lại không được ngắn hơn thời gian chờ lưu trữ',
  'Object storage settings saved': 'Đã lưu cài đặt lưu trữ đối tượng',
  'Failed to save object storage settings':
    'Không thể lưu cài đặt lưu trữ đối tượng',
  'Save settings': 'Lưu cài đặt',
  'Secret storage notice': 'Lưu ý về lưu trữ bí mật',
  'The S3 Secret Access Key is stored as plaintext in the database. Anyone with database or backup access can read it.':
    'S3 Secret Access Key được lưu dạng văn bản thuần trong cơ sở dữ liệu; người có quyền đọc cơ sở dữ liệu hoặc bản sao lưu có thể xem.',
  'S3 connection': 'Kết nối S3',
  'Objects remain private and are delivered with temporary signed URLs.':
    'Đối tượng vẫn ở chế độ riêng tư và được cung cấp qua URL ký tạm thời.',
  'Secret configured': 'Đã cấu hình bí mật',
  'Secret not configured': 'Chưa cấu hình bí mật',
  'Endpoint URL': 'URL điểm cuối',
  'Leave blank to use the standard AWS endpoint.':
    'Để trống để dùng điểm cuối AWS tiêu chuẩn.',
  'Secret Access Key': 'Secret Access Key',
  'Leave blank to keep the existing secret': 'Để trống để giữ bí mật hiện tại',
  'Request parameters were not retained for this historical task':
    'Các tham số yêu cầu không được lưu cho tác vụ cũ này',
  'Retry selected uploads': 'Thử lại các tệp đã chọn',
  'Retry all failed image uploads': 'Thử lại toàn bộ ảnh tải lên thất bại',
  'Retry selected': 'Thử lại mục đã chọn',
  'Retry all failed': 'Thử lại toàn bộ lỗi',
  'Retry all failed uploads?': 'Thử lại mọi tệp tải lên thất bại?',
  'Only persistent staged files are uploaded again. Image generation and billing are not repeated.':
    'Chỉ tải lại tệp tạm bền vững; không tạo lại ảnh hoặc tính phí lại.',
  'Start retry': 'Bắt đầu thử lại',
  'Generation status': 'Trạng thái tạo ảnh',
  'Output availability': 'Khả dụng đầu ra',
  'Billing status': 'Trạng thái tính phí',
  Objects: 'Đối tượng',
  Attempts: 'Số lần thử',
  'Staging integrity': 'Tính toàn vẹn tệp tạm',
  'Created at': 'Thời gian tạo',
  'Select retryable tasks': 'Chọn tác vụ có thể thử lại',
  'Select task {{taskId}}': 'Chọn tác vụ {{taskId}}',
  'No async image tasks found': 'Không tìm thấy tác vụ ảnh bất đồng bộ',
  '{{count}} tasks': '{{count}} tác vụ',
  'Accepted {{accepted}} tasks; skipped {{skipped}} tasks':
    'Đã nhận {{accepted}} tác vụ; bỏ qua {{skipped}} tác vụ',
  'Failed to retry image uploads': 'Không thể thử lại tải ảnh',
  'Bulk retry operation started: {{operationId}}':
    'Đã bắt đầu thử lại hàng loạt: {{operationId}}',
  'Failed to start bulk retry': 'Không thể bắt đầu thử lại hàng loạt',
  queued: 'Đang chờ',
  archiving: 'Đang lưu trữ',
  available: 'Khả dụng',
  reserved: 'Đã giữ trước',
  settled: 'Đã quyết toán',
  refunded: 'Đã hoàn tiền',
  not_staged: 'Chưa lưu tạm',
})

Object.assign(newKeys.en, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'OpenAI Video / Sora Compatible',
  'Seedance Compatible': 'Seedance Compatible',
  'Select video protocol': 'Select video protocol',
  'Video Protocol': 'Video Protocol',
})
Object.assign(newKeys.zh, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'OpenAI Video / Sora 兼容',
  'Seedance Compatible': 'Seedance 兼容',
  'Select video protocol': '选择视频协议',
  'Video Protocol': '视频协议',
})
Object.assign(newKeys['zh-TW'], {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'OpenAI Video / Sora 相容',
  'Seedance Compatible': 'Seedance 相容',
  'Select video protocol': '選擇影片協議',
  'Video Protocol': '影片協議',
})
Object.assign(newKeys.fr, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'Compatible OpenAI Video / Sora',
  'Seedance Compatible': 'Compatible Seedance',
  'Select video protocol': 'Sélectionner le protocole vidéo',
  'Video Protocol': 'Protocole vidéo',
})
Object.assign(newKeys.ja, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'OpenAI Video / Sora 互換',
  'Seedance Compatible': 'Seedance 互換',
  'Select video protocol': '動画プロトコルを選択',
  'Video Protocol': '動画プロトコル',
})
Object.assign(newKeys.ru, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'Совместимый с OpenAI Video / Sora',
  'Seedance Compatible': 'Совместимый с Seedance',
  'Select video protocol': 'Выберите видеопротокол',
  'Video Protocol': 'Видеопротокол',
})
Object.assign(newKeys.vi, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'Tương thích OpenAI Video / Sora',
  'Seedance Compatible': 'Tương thích Seedance',
  'Select video protocol': 'Chọn giao thức video',
  'Video Protocol': 'Giao thức video',
})

async function main() {
  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))
    let changed = false
    for (const [key, value] of Object.entries(trans)) {
      if (json.translation[key] !== value) {
        json.translation[key] = value
        changed = true
      }
    }
    if (changed) {
      json.translation = Object.fromEntries(
        Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
      )
      await fs.writeFile(filePath, stableStringify(json))
    }
  }
}

await main()
