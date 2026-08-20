/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return `${JSON.stringify(obj, null, 2).replaceAll(
    '\"footer.newapi.projectAttributionSuffix\":',
    '\"footer.new\\\\u0061pi.projectAttributionSuffix\":'
  )}\n`
}

const newKeys = {
  en: {
    'Archive completed videos to the configured video object storage and return presigned URLs':
      'Archive completed videos to the configured video object storage and return presigned URLs',
    'Archived videos remain private and are delivered with temporary signed URLs.':
      'Archived videos remain private and are delivered with temporary signed URLs.',
    'Async images': 'Async images',
    'Business ID': 'Business ID',
    'Business ID is required': 'Business ID is required',
    'Failed to save video object storage settings':
      'Failed to save video object storage settings',
    'Keep using the upstream video content URL':
      'Keep using the upstream video content URL',
    'Object storage type': 'Object storage type',
    'Store video files in S3': 'Store video files in S3',
    'The root prefix before user-files.': 'The root prefix before user-files.',
    'Use one path segment without slashes or spaces':
      'Use one path segment without slashes or spaces',
    'Used as the business path segment and object namespace.':
      'Used as the business path segment and object namespace.',
    'Video object storage settings saved':
      'Video object storage settings saved',
    'Video S3 connection': 'Video S3 connection',
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
    'Archive completed videos to the configured video object storage and return presigned URLs':
      '将已完成的视频归档到配置的视频对象存储，并返回签名 URL',
    'Archived videos remain private and are delivered with temporary signed URLs.':
      '归档后的视频保持私有，并通过临时签名 URL 提供访问。',
    'Async images': '异步图片',
    'Business ID': '业务 ID',
    'Business ID is required': '业务 ID 不能为空',
    'Failed to save video object storage settings': '保存视频对象存储设置失败',
    'Keep using the upstream video content URL': '继续使用上游视频内容地址',
    'Object storage type': '对象存储类型',
    'Store video files in S3': '将视频文件存储到 S3',
    'The root prefix before user-files.': 'user-files 之前的根前缀。',
    'Use one path segment without slashes or spaces':
      '请使用不含斜杠和空格的单个路径段',
    'Used as the business path segment and object namespace.':
      '用作业务路径段和对象命名空间。',
    'Video object storage settings saved': '视频对象存储设置已保存',
    'Video S3 connection': '视频 S3 连接',
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
    'Archive completed videos to the configured video object storage and return presigned URLs':
      '將已完成的影片封存到設定的影片物件儲存，並傳回簽名 URL',
    'Archived videos remain private and are delivered with temporary signed URLs.':
      '封存後的影片保持私有，並透過臨時簽名 URL 提供存取。',
    'Async images': '非同步圖片',
    'Business ID': '業務 ID',
    'Business ID is required': '業務 ID 為必填',
    'Failed to save video object storage settings': '儲存影片物件儲存設定失敗',
    'Keep using the upstream video content URL': '繼續使用上游影片內容位址',
    'Object storage type': '物件儲存類型',
    'Store video files in S3': '將影片檔案儲存到 S3',
    'The root prefix before user-files.': 'user-files 之前的根前綴。',
    'Use one path segment without slashes or spaces':
      '請使用不含斜線和空格的單一路徑段',
    'Used as the business path segment and object namespace.':
      '用作業務路徑段和物件命名空間。',
    'Video object storage settings saved': '影片物件儲存設定已儲存',
    'Video S3 connection': '影片 S3 連線',
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
    'Archive completed videos to the configured video object storage and return presigned URLs':
      'Archiver les vidéos terminées dans le stockage configuré et renvoyer des URL présignées',
    'Archived videos remain private and are delivered with temporary signed URLs.':
      'Les vidéos archivées restent privées et sont accessibles par des URL signées temporaires.',
    'Async images': 'Images asynchrones',
    'Business ID': 'ID métier',
    'Business ID is required': 'L’ID métier est requis',
    'Failed to save video object storage settings':
      'Échec de l’enregistrement du stockage vidéo',
    'Keep using the upstream video content URL':
      'Continuer à utiliser l’URL vidéo du fournisseur',
    'Object storage type': 'Type de stockage objet',
    'Store video files in S3': 'Stocker les vidéos dans S3',
    'The root prefix before user-files.':
      'Préfixe racine placé avant user-files.',
    'Use one path segment without slashes or spaces':
      'Utilisez un seul segment sans barre oblique ni espace',
    'Used as the business path segment and object namespace.':
      'Utilisé comme segment métier et espace de noms des objets.',
    'Video object storage settings saved':
      'Paramètres du stockage vidéo enregistrés',
    'Video S3 connection': 'Connexion S3 vidéo',
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
    'Archive completed videos to the configured video object storage and return presigned URLs':
      '完了した動画を設定済みの動画オブジェクトストレージに保存し、署名付き URL を返します',
    'Archived videos remain private and are delivered with temporary signed URLs.':
      '保存された動画は非公開のまま、一時的な署名付き URL で配信されます。',
    'Async images': '非同期画像',
    'Business ID': 'ビジネス ID',
    'Business ID is required': 'ビジネス ID は必須です',
    'Failed to save video object storage settings':
      '動画オブジェクトストレージ設定を保存できませんでした',
    'Keep using the upstream video content URL':
      'アップストリームの動画 URL を引き続き使用します',
    'Object storage type': 'オブジェクトストレージの種類',
    'Store video files in S3': '動画ファイルを S3 に保存',
    'The root prefix before user-files.':
      'user-files より前に付けるルートプレフィックスです。',
    'Use one path segment without slashes or spaces':
      'スラッシュや空白を含まない単一のパスセグメントを使用してください',
    'Used as the business path segment and object namespace.':
      'ビジネス用パスセグメントとオブジェクト名前空間として使用します。',
    'Video object storage settings saved':
      '動画オブジェクトストレージ設定を保存しました',
    'Video S3 connection': '動画 S3 接続',
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
    'Archive completed videos to the configured video object storage and return presigned URLs':
      'Архивировать готовые видео в настроенное объектное хранилище и возвращать подписанные URL',
    'Archived videos remain private and are delivered with temporary signed URLs.':
      'Архивные видео остаются закрытыми и доступны по временным подписанным URL.',
    'Async images': 'Асинхронные изображения',
    'Business ID': 'Идентификатор бизнеса',
    'Business ID is required': 'Требуется идентификатор бизнеса',
    'Failed to save video object storage settings':
      'Не удалось сохранить настройки хранилища видео',
    'Keep using the upstream video content URL':
      'Продолжать использовать URL видео поставщика',
    'Object storage type': 'Тип объектного хранилища',
    'Store video files in S3': 'Хранить видеофайлы в S3',
    'The root prefix before user-files.': 'Корневой префикс перед user-files.',
    'Use one path segment without slashes or spaces':
      'Используйте один сегмент пути без косых черт и пробелов',
    'Used as the business path segment and object namespace.':
      'Используется как сегмент бизнес-пути и пространство имён объектов.',
    'Video object storage settings saved':
      'Настройки хранилища видео сохранены',
    'Video S3 connection': 'Подключение S3 для видео',
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
    'Archive completed videos to the configured video object storage and return presigned URLs':
      'Lưu trữ video đã hoàn tất vào kho đối tượng video đã cấu hình và trả về URL đã ký',
    'Archived videos remain private and are delivered with temporary signed URLs.':
      'Video đã lưu trữ vẫn ở chế độ riêng tư và được cung cấp qua URL đã ký tạm thời.',
    'Async images': 'Hình ảnh bất đồng bộ',
    'Business ID': 'ID nghiệp vụ',
    'Business ID is required': 'Bắt buộc có ID nghiệp vụ',
    'Failed to save video object storage settings':
      'Không thể lưu cài đặt kho đối tượng video',
    'Keep using the upstream video content URL':
      'Tiếp tục dùng URL nội dung video từ nhà cung cấp',
    'Object storage type': 'Loại kho đối tượng',
    'Store video files in S3': 'Lưu tệp video vào S3',
    'The root prefix before user-files.': 'Tiền tố gốc đứng trước user-files.',
    'Use one path segment without slashes or spaces':
      'Dùng một đoạn đường dẫn không có dấu gạch chéo hoặc khoảng trắng',
    'Used as the business path segment and object namespace.':
      'Được dùng làm đoạn đường dẫn nghiệp vụ và không gian tên đối tượng.',
    'Video object storage settings saved': 'Đã lưu cài đặt kho đối tượng video',
    'Video S3 connection': 'Kết nối S3 cho video',
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
  'S3 object key prefix': 'S3 object key prefix',
  'S3 object key prefix is required': 'S3 object key prefix is required',
  'Use a relative S3 prefix without empty, . or .. path segments':
    'Use a relative S3 prefix without empty, . or .. path segments',
  'New async image objects are stored under this prefix.':
    'New async image objects are stored under this prefix.',
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
  'S3 object key prefix': 'S3 对象键前缀',
  'S3 object key prefix is required': 'S3 对象键前缀不能为空',
  'Use a relative S3 prefix without empty, . or .. path segments':
    '请使用不含空路径段、. 或 .. 的相对 S3 前缀',
  'New async image objects are stored under this prefix.':
    '新的异步图片对象将存储在此前缀下。',
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
  'S3 object key prefix': 'S3 物件鍵前綴',
  'S3 object key prefix is required': 'S3 物件鍵前綴為必填',
  'Use a relative S3 prefix without empty, . or .. path segments':
    '請使用不含空路徑段、. 或 .. 的相對 S3 前綴',
  'New async image objects are stored under this prefix.':
    '新的非同步圖片物件將儲存在此前綴下。',
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
  'S3 object key prefix': 'Préfixe de clé d’objet S3',
  'S3 object key prefix is required': 'Le préfixe de clé S3 est obligatoire',
  'Use a relative S3 prefix without empty, . or .. path segments':
    'Utilisez un préfixe S3 relatif sans segment vide, . ou ..',
  'New async image objects are stored under this prefix.':
    'Les nouveaux objets d’image asynchrones sont stockés sous ce préfixe.',
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
  'S3 object key prefix': 'S3 オブジェクトキープレフィックス',
  'S3 object key prefix is required':
    'S3 オブジェクトキープレフィックスは必須です',
  'Use a relative S3 prefix without empty, . or .. path segments':
    '空のセグメント、.、.. を含まない相対 S3 プレフィックスを使用してください',
  'New async image objects are stored under this prefix.':
    '新しい非同期画像オブジェクトはこのプレフィックス配下に保存されます。',
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
  'S3 object key prefix': 'Префикс ключа объекта S3',
  'S3 object key prefix is required': 'Префикс ключа объекта S3 обязателен',
  'Use a relative S3 prefix without empty, . or .. path segments':
    'Используйте относительный префикс S3 без пустых сегментов, . и ..',
  'New async image objects are stored under this prefix.':
    'Новые объекты асинхронных изображений сохраняются под этим префиксом.',
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
  'S3 object key prefix': 'Tiền tố khóa đối tượng S3',
  'S3 object key prefix is required': 'Bắt buộc nhập tiền tố khóa đối tượng S3',
  'Use a relative S3 prefix without empty, . or .. path segments':
    'Dùng tiền tố S3 tương đối không có đoạn rỗng, . hoặc ..',
  'New async image objects are stored under this prefix.':
    'Các đối tượng ảnh bất đồng bộ mới được lưu dưới tiền tố này.',
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
  'Redemption time': 'Redemption time',
  megabyai: 'megabyai',
  'Select video protocol': 'Select video protocol',
  'Video Protocol': 'Video Protocol',
})
Object.assign(newKeys.zh, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'OpenAI Video / Sora 兼容',
  'Redemption time': '兑换时间',
  megabyai: 'megabyai',
  'Select video protocol': '选择视频协议',
  'Video Protocol': '视频协议',
})
Object.assign(newKeys['zh-TW'], {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'OpenAI Video / Sora 相容',
  'Redemption time': '兌換時間',
  megabyai: 'megabyai',
  'Select video protocol': '選擇影片協議',
  'Video Protocol': '影片協議',
})
Object.assign(newKeys.fr, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'Compatible OpenAI Video / Sora',
  'Redemption time': "Date d'échange",
  megabyai: 'megabyai',
  'Select video protocol': 'Sélectionner le protocole vidéo',
  'Video Protocol': 'Protocole vidéo',
})
Object.assign(newKeys.ja, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'OpenAI Video / Sora 互換',
  'Redemption time': '引き換え日時',
  megabyai: 'megabyai',
  'Select video protocol': '動画プロトコルを選択',
  'Video Protocol': '動画プロトコル',
})
Object.assign(newKeys.ru, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'Совместимый с OpenAI Video / Sora',
  'Redemption time': 'Время активации',
  megabyai: 'megabyai',
  'Select video protocol': 'Выберите видеопротокол',
  'Video Protocol': 'Видеопротокол',
})
Object.assign(newKeys.vi, {
  'Agnes Video V2': 'Agnes Video V2',
  'OpenAI Video / Sora Compatible': 'Tương thích OpenAI Video / Sora',
  'Redemption time': 'Thời gian đổi mã',
  megabyai: 'megabyai',
  'Select video protocol': 'Chọn giao thức video',
  'Video Protocol': 'Giao thức video',
})

Object.assign(newKeys.en, {
  'Add a resolution, for example 720p': 'Add a resolution, for example 720p',
  'Configure each upstream video model and its reference media limits':
    'Configure each upstream video model and its reference media limits',
  'Configure at least one resolution': 'Configure at least one resolution',
  'Configure at least one video model': 'Configure at least one video model',
  'Duration must be a whole number': 'Duration must be a whole number',
  'Duration must be between 1 and 3600 seconds':
    'Duration must be between 1 and 3600 seconds',
  'Fetch or add channel models before configuring video models':
    'Fetch or add channel models before configuring video models',
  'Max reference audios': 'Max reference audios',
  'Max reference images': 'Max reference images',
  'Max reference videos': 'Max reference videos',
  'Maximum duration (seconds)': 'Maximum duration (seconds)',
  'Maximum duration is required': 'Maximum duration is required',
  'Minimum duration (seconds)': 'Minimum duration (seconds)',
  'Minimum duration cannot exceed maximum duration':
    'Minimum duration cannot exceed maximum duration',
  'Minimum duration is required': 'Minimum duration is required',
  'No video model capabilities configured':
    'No video model capabilities configured',
  'Press Enter or comma to add resolutions':
    'Press Enter or comma to add resolutions',
  'Select upstream model ID': 'Select upstream model ID',
  'Video model capabilities': 'Video model capabilities',
})
Object.assign(newKeys.zh, {
  'Add a resolution, for example 720p': '添加分辨率，例如 720p',
  'Configure each upstream video model and its reference media limits':
    '配置每个上游视频模型及其参考素材数量上限',
  'Configure at least one resolution': '至少配置一个分辨率',
  'Configure at least one video model': '至少配置一个视频模型',
  'Duration must be a whole number': '时长必须为整数',
  'Duration must be between 1 and 3600 seconds': '时长必须在 1 到 3600 秒之间',
  'Fetch or add channel models before configuring video models':
    '请先获取或添加渠道模型，再配置视频模型',
  'Max reference audios': '参考音频上限',
  'Max reference images': '参考图片上限',
  'Max reference videos': '参考视频上限',
  'Maximum duration (seconds)': '最大时长（秒）',
  'Maximum duration is required': '必须配置最大时长',
  'Minimum duration (seconds)': '最小时长（秒）',
  'Minimum duration cannot exceed maximum duration': '最小时长不能大于最大时长',
  'Minimum duration is required': '必须配置最小时长',
  'No video model capabilities configured': '尚未配置视频模型能力',
  'Press Enter or comma to add resolutions': '按回车键或逗号添加分辨率',
  'Select upstream model ID': '选择上游模型 ID',
  'Video model capabilities': '视频模型能力',
})
Object.assign(newKeys['zh-TW'], {
  'Add a resolution, for example 720p': '新增解析度，例如 720p',
  'Configure each upstream video model and its reference media limits':
    '設定每個上游影片模型及其參考素材數量上限',
  'Configure at least one resolution': '至少設定一個解析度',
  'Configure at least one video model': '至少設定一個影片模型',
  'Duration must be a whole number': '時長必須為整數',
  'Duration must be between 1 and 3600 seconds':
    '時長必須介於 1 到 3600 秒之間',
  'Fetch or add channel models before configuring video models':
    '請先取得或新增渠道模型，再設定影片模型',
  'Max reference audios': '參考音訊上限',
  'Max reference images': '參考圖片上限',
  'Max reference videos': '參考影片上限',
  'Maximum duration (seconds)': '最長時長（秒）',
  'Maximum duration is required': '必須設定最長時長',
  'Minimum duration (seconds)': '最短時長（秒）',
  'Minimum duration cannot exceed maximum duration': '最短時長不能大於最長時長',
  'Minimum duration is required': '必須設定最短時長',
  'No video model capabilities configured': '尚未設定影片模型能力',
  'Press Enter or comma to add resolutions': '按 Enter 或逗號新增解析度',
  'Select upstream model ID': '選擇上游模型 ID',
  'Video model capabilities': '影片模型能力',
})
Object.assign(newKeys.fr, {
  'Add a resolution, for example 720p': 'Ajouter une résolution, par ex. 720p',
  'Configure each upstream video model and its reference media limits':
    'Configurer chaque modèle vidéo en amont et ses limites de médias de référence',
  'Configure at least one resolution': 'Configurez au moins une résolution',
  'Configure at least one video model': 'Configurez au moins un modèle vidéo',
  'Duration must be a whole number': 'La durée doit être un nombre entier',
  'Duration must be between 1 and 3600 seconds':
    'La durée doit être comprise entre 1 et 3 600 secondes',
  'Fetch or add channel models before configuring video models':
    'Récupérez ou ajoutez les modèles du canal avant de configurer les modèles vidéo',
  'Max reference audios': 'Audios de référence max.',
  'Max reference images': 'Images de référence max.',
  'Max reference videos': 'Vidéos de référence max.',
  'Maximum duration (seconds)': 'Durée maximale (secondes)',
  'Maximum duration is required': 'La durée maximale est requise',
  'Minimum duration (seconds)': 'Durée minimale (secondes)',
  'Minimum duration cannot exceed maximum duration':
    'La durée minimale ne peut pas dépasser la durée maximale',
  'Minimum duration is required': 'La durée minimale est requise',
  'No video model capabilities configured':
    'Aucune capacité de modèle vidéo configurée',
  'Press Enter or comma to add resolutions':
    'Appuyez sur Entrée ou une virgule pour ajouter des résolutions',
  'Select upstream model ID': "Sélectionner l'ID du modèle en amont",
  'Video model capabilities': 'Capacités des modèles vidéo',
})
Object.assign(newKeys.ja, {
  'Add a resolution, for example 720p': '解像度を追加（例: 720p）',
  'Configure each upstream video model and its reference media limits':
    '上流動画モデルごとに参照メディアの上限を設定します',
  'Configure at least one resolution': '解像度を1つ以上設定してください',
  'Configure at least one video model': '動画モデルを1つ以上設定してください',
  'Duration must be a whole number': '時間は整数で指定してください',
  'Duration must be between 1 and 3600 seconds':
    '時間は1〜3600秒の範囲で指定してください',
  'Fetch or add channel models before configuring video models':
    '動画モデルを設定する前にチャネルモデルを取得または追加してください',
  'Max reference audios': '参照音声の上限',
  'Max reference images': '参照画像の上限',
  'Max reference videos': '参照動画の上限',
  'Maximum duration (seconds)': '最大時間（秒）',
  'Maximum duration is required': '最大時間を設定してください',
  'Minimum duration (seconds)': '最小時間（秒）',
  'Minimum duration cannot exceed maximum duration':
    '最小時間は最大時間以下にしてください',
  'Minimum duration is required': '最小時間を設定してください',
  'No video model capabilities configured': '動画モデル機能が未設定です',
  'Press Enter or comma to add resolutions':
    'Enter またはカンマで解像度を追加します',
  'Select upstream model ID': '上流モデル ID を選択',
  'Video model capabilities': '動画モデル機能',
})
Object.assign(newKeys.ru, {
  'Add a resolution, for example 720p': 'Добавьте разрешение, например 720p',
  'Configure each upstream video model and its reference media limits':
    'Настройте каждую вышестоящую видеомодель и лимиты референсных материалов',
  'Configure at least one resolution': 'Настройте хотя бы одно разрешение',
  'Configure at least one video model': 'Настройте хотя бы одну видеомодель',
  'Duration must be a whole number': 'Длительность должна быть целым числом',
  'Duration must be between 1 and 3600 seconds':
    'Длительность должна быть от 1 до 3600 секунд',
  'Fetch or add channel models before configuring video models':
    'Получите или добавьте модели канала перед настройкой видеомоделей',
  'Max reference audios': 'Макс. референсных аудио',
  'Max reference images': 'Макс. референсных изображений',
  'Max reference videos': 'Макс. референсных видео',
  'Maximum duration (seconds)': 'Максимальная длительность (сек.)',
  'Maximum duration is required': 'Укажите максимальную длительность',
  'Minimum duration (seconds)': 'Минимальная длительность (сек.)',
  'Minimum duration cannot exceed maximum duration':
    'Минимальная длительность не может превышать максимальную',
  'Minimum duration is required': 'Укажите минимальную длительность',
  'No video model capabilities configured':
    'Возможности видеомоделей не настроены',
  'Press Enter or comma to add resolutions':
    'Нажмите Enter или запятую, чтобы добавить разрешение',
  'Select upstream model ID': 'Выберите ID вышестоящей модели',
  'Video model capabilities': 'Возможности видеомоделей',
})
Object.assign(newKeys.vi, {
  'Add a resolution, for example 720p': 'Thêm độ phân giải, ví dụ 720p',
  'Configure each upstream video model and its reference media limits':
    'Cấu hình từng mô hình video thượng nguồn và giới hạn nội dung tham chiếu',
  'Configure at least one resolution': 'Cấu hình ít nhất một độ phân giải',
  'Configure at least one video model': 'Cấu hình ít nhất một mô hình video',
  'Duration must be a whole number': 'Thời lượng phải là số nguyên',
  'Duration must be between 1 and 3600 seconds':
    'Thời lượng phải từ 1 đến 3600 giây',
  'Fetch or add channel models before configuring video models':
    'Tải hoặc thêm mô hình kênh trước khi cấu hình mô hình video',
  'Max reference audios': 'Âm thanh tham chiếu tối đa',
  'Max reference images': 'Ảnh tham chiếu tối đa',
  'Max reference videos': 'Video tham chiếu tối đa',
  'Maximum duration (seconds)': 'Thời lượng tối đa (giây)',
  'Maximum duration is required': 'Bắt buộc cấu hình thời lượng tối đa',
  'Minimum duration (seconds)': 'Thời lượng tối thiểu (giây)',
  'Minimum duration cannot exceed maximum duration':
    'Thời lượng tối thiểu không được lớn hơn thời lượng tối đa',
  'Minimum duration is required': 'Bắt buộc cấu hình thời lượng tối thiểu',
  'No video model capabilities configured':
    'Chưa cấu hình khả năng mô hình video',
  'Press Enter or comma to add resolutions':
    'Nhấn Enter hoặc dấu phẩy để thêm độ phân giải',
  'Select upstream model ID': 'Chọn ID mô hình thượng nguồn',
  'Video model capabilities': 'Khả năng mô hình video',
})

Object.assign(newKeys.en, {
  'Failed to retry video upload': 'Failed to retry video upload',
  'Retry video upload': 'Retry video upload',
  'Storage error': 'Storage error',
  Uploading: 'Uploading',
  'Video delivery priority': 'Video delivery priority',
  'Video upload retry started': 'Video upload retry started',
})
Object.assign(newKeys.zh, {
  'Failed to retry video upload': '视频上传重试失败',
  'Retry video upload': '重试视频上传',
  'Storage error': '存储错误',
  Uploading: '上传中',
  'Video delivery priority': '视频交付优先级',
  'Video upload retry started': '已开始重试视频上传',
})
Object.assign(newKeys['zh-TW'], {
  'Failed to retry video upload': '影片上傳重試失敗',
  'Retry video upload': '重試影片上傳',
  'Storage error': '儲存錯誤',
  Uploading: '上傳中',
  'Video delivery priority': '影片交付優先順序',
  'Video upload retry started': '已開始重試影片上傳',
})
Object.assign(newKeys.fr, {
  'Failed to retry video upload':
    'Échec de la nouvelle tentative de téléversement vidéo',
  'Retry video upload': 'Réessayer le téléversement',
  'Storage error': 'Erreur de stockage',
  Uploading: 'Téléversement en cours',
  'Video delivery priority': 'Priorité de livraison vidéo',
  'Video upload retry started': 'Nouvelle tentative de téléversement lancée',
})
Object.assign(newKeys.ja, {
  'Failed to retry video upload': '動画アップロードの再試行に失敗しました',
  'Retry video upload': '動画アップロードを再試行',
  'Storage error': 'ストレージエラー',
  Uploading: 'アップロード中',
  'Video delivery priority': '動画配信の優先順位',
  'Video upload retry started': '動画アップロードの再試行を開始しました',
})
Object.assign(newKeys.ru, {
  'Failed to retry video upload': 'Не удалось повторить загрузку видео',
  'Retry video upload': 'Повторить загрузку видео',
  'Storage error': 'Ошибка хранилища',
  Uploading: 'Загрузка',
  'Video delivery priority': 'Приоритет доставки видео',
  'Video upload retry started': 'Повторная загрузка видео запущена',
})
Object.assign(newKeys.vi, {
  'Failed to retry video upload': 'Không thể thử lại việc tải video lên',
  'Retry video upload': 'Thử tải lại video',
  'Storage error': 'Lỗi lưu trữ',
  Uploading: 'Đang tải lên',
  'Video delivery priority': 'Ưu tiên phân phối video',
  'Video upload retry started': 'Đã bắt đầu thử tải lại video',
})

Object.assign(newKeys.en, {
  'S3 Preferred': 'S3 Preferred',
  'Select all uploadable video tasks': 'Select all uploadable video tasks',
  'Select video task': 'Select video task',
  'Upload to S3': 'Upload to S3',
  'Video upload started': 'Video upload started',
  'Video uploads started': 'Video uploads started',
  'Failed to upload video to S3': 'Failed to upload video to S3',
  'Failed to upload videos to S3': 'Failed to upload videos to S3',
  'Not uploaded': 'Not uploaded',
  '{{count}} task(s) were skipped': '{{count}} task(s) were skipped',
})
Object.assign(newKeys.zh, {
  'S3 Preferred': 'S3 优先',
  'Select all uploadable video tasks': '选择全部可上传的视频任务',
  'Select video task': '选择视频任务',
  'Upload to S3': '上传到 S3',
  'Video upload started': '已开始上传视频',
  'Video uploads started': '已开始批量上传视频',
  'Failed to upload video to S3': '视频上传到 S3 失败',
  'Failed to upload videos to S3': '批量上传视频到 S3 失败',
  'Not uploaded': '未上传',
  '{{count}} task(s) were skipped': '已跳过 {{count}} 个任务',
})
Object.assign(newKeys['zh-TW'], {
  'S3 Preferred': 'S3 優先',
  'Select all uploadable video tasks': '選擇全部可上傳的影片任務',
  'Select video task': '選擇影片任務',
  'Upload to S3': '上傳到 S3',
  'Video upload started': '已開始上傳影片',
  'Video uploads started': '已開始批次上傳影片',
  'Failed to upload video to S3': '影片上傳到 S3 失敗',
  'Failed to upload videos to S3': '批次上傳影片到 S3 失敗',
  'Not uploaded': '未上傳',
  '{{count}} task(s) were skipped': '已略過 {{count}} 個任務',
})
Object.assign(newKeys.fr, {
  'S3 Preferred': 'S3 prioritaire',
  'Select all uploadable video tasks':
    'Sélectionner toutes les tâches vidéo téléversables',
  'Select video task': 'Sélectionner la tâche vidéo',
  'Upload to S3': 'Téléverser vers S3',
  'Video upload started': 'Téléversement vidéo lancé',
  'Video uploads started': 'Téléversements vidéo lancés',
  'Failed to upload video to S3': 'Échec du téléversement vidéo vers S3',
  'Failed to upload videos to S3': 'Échec du téléversement des vidéos vers S3',
  'Not uploaded': 'Non téléversé',
  '{{count}} task(s) were skipped': '{{count}} tâche(s) ignorée(s)',
})
Object.assign(newKeys.ja, {
  'S3 Preferred': 'S3 を優先',
  'Select all uploadable video tasks':
    'アップロード可能な動画タスクをすべて選択',
  'Select video task': '動画タスクを選択',
  'Upload to S3': 'S3 にアップロード',
  'Video upload started': '動画のアップロードを開始しました',
  'Video uploads started': '動画の一括アップロードを開始しました',
  'Failed to upload video to S3': '動画を S3 にアップロードできませんでした',
  'Failed to upload videos to S3':
    '動画を S3 に一括アップロードできませんでした',
  'Not uploaded': '未アップロード',
  '{{count}} task(s) were skipped': '{{count}} 件のタスクをスキップしました',
})
Object.assign(newKeys.ru, {
  'S3 Preferred': 'Предпочитать S3',
  'Select all uploadable video tasks':
    'Выбрать все доступные для загрузки видеозадачи',
  'Select video task': 'Выбрать видеозадачу',
  'Upload to S3': 'Загрузить в S3',
  'Video upload started': 'Загрузка видео начата',
  'Video uploads started': 'Загрузка видео начата',
  'Failed to upload video to S3': 'Не удалось загрузить видео в S3',
  'Failed to upload videos to S3': 'Не удалось загрузить видео в S3',
  'Not uploaded': 'Не загружено',
  '{{count}} task(s) were skipped': 'Пропущено задач: {{count}}',
})
Object.assign(newKeys.vi, {
  'S3 Preferred': 'Ưu tiên S3',
  'Select all uploadable video tasks':
    'Chọn tất cả tác vụ video có thể tải lên',
  'Select video task': 'Chọn tác vụ video',
  'Upload to S3': 'Tải lên S3',
  'Video upload started': 'Đã bắt đầu tải video lên',
  'Video uploads started': 'Đã bắt đầu tải các video lên',
  'Failed to upload video to S3': 'Không thể tải video lên S3',
  'Failed to upload videos to S3': 'Không thể tải các video lên S3',
  'Not uploaded': 'Chưa tải lên',
  '{{count}} task(s) were skipped': 'Đã bỏ qua {{count}} tác vụ',
})

Object.assign(newKeys.en, {
  'Upstream HTTP diagnostics': 'Upstream HTTP diagnostics',
  'Task submission': 'Task submission',
  'Failed task polling': 'Failed task polling',
  'Task polling': 'Task polling',
  'No upstream HTTP diagnostics recorded':
    'No upstream HTTP diagnostics recorded',
  'Content truncated at 64 KiB': 'Content truncated at 64 KiB',
  'Transport error': 'Transport error',
})
Object.assign(newKeys.zh, {
  'Upstream HTTP diagnostics': '上游 HTTP 诊断',
  'Task submission': '提交任务',
  'Failed task polling': '失败任务轮询',
  'Task polling': '任务轮询',
  'No upstream HTTP diagnostics recorded': '未记录上游 HTTP 诊断信息',
  'Content truncated at 64 KiB': '内容已在 64 KiB 处截断',
  'Transport error': '传输错误',
})
Object.assign(newKeys['zh-TW'], {
  'Upstream HTTP diagnostics': '上游 HTTP 診斷',
  'Task submission': '提交任務',
  'Failed task polling': '失敗任務輪詢',
  'Task polling': '任務輪詢',
  'No upstream HTTP diagnostics recorded': '未記錄上游 HTTP 診斷資訊',
  'Content truncated at 64 KiB': '內容已在 64 KiB 處截斷',
  'Transport error': '傳輸錯誤',
})
Object.assign(newKeys.fr, {
  'Upstream HTTP diagnostics': 'Diagnostic HTTP en amont',
  'Task submission': 'Envoi de la tâche',
  'Failed task polling': 'Interrogation de la tâche échouée',
  'Task polling': 'Interrogation de la tâche',
  'No upstream HTTP diagnostics recorded':
    'Aucun diagnostic HTTP en amont enregistré',
  'Content truncated at 64 KiB': 'Contenu tronqué à 64 Kio',
  'Transport error': 'Erreur de transport',
})
Object.assign(newKeys.ja, {
  'Upstream HTTP diagnostics': '上流 HTTP 診断',
  'Task submission': 'タスク送信',
  'Failed task polling': '失敗タスクのポーリング',
  'Task polling': 'タスクのポーリング',
  'No upstream HTTP diagnostics recorded': '上流 HTTP 診断は記録されていません',
  'Content truncated at 64 KiB': 'コンテンツは 64 KiB で切り詰められました',
  'Transport error': '通信エラー',
})
Object.assign(newKeys.ru, {
  'Upstream HTTP diagnostics': 'Диагностика HTTP вышестоящего сервиса',
  'Task submission': 'Отправка задачи',
  'Failed task polling': 'Опрос завершившейся с ошибкой задачи',
  'Task polling': 'Опрос задачи',
  'No upstream HTTP diagnostics recorded':
    'Диагностика HTTP вышестоящего сервиса не записана',
  'Content truncated at 64 KiB': 'Содержимое обрезано до 64 КиБ',
  'Transport error': 'Ошибка передачи',
})
Object.assign(newKeys.vi, {
  'Upstream HTTP diagnostics': 'Chẩn đoán HTTP thượng nguồn',
  'Task submission': 'Gửi tác vụ',
  'Failed task polling': 'Truy vấn tác vụ thất bại',
  'Task polling': 'Truy vấn trạng thái tác vụ',
  'No upstream HTTP diagnostics recorded':
    'Chưa ghi nhận chẩn đoán HTTP thượng nguồn',
  'Content truncated at 64 KiB': 'Nội dung đã bị cắt tại 64 KiB',
  'Transport error': 'Lỗi truyền tải',
})

Object.assign(newKeys.en, {
  megabyai: 'megabyai',
  'Supported aspect ratios': 'Supported aspect ratios',
  'Add an aspect ratio, for example 16:9':
    'Add an aspect ratio, for example 16:9',
  'Press Enter or comma to add aspect ratios':
    'Press Enter or comma to add aspect ratios',
  'Supports native audio': 'Supports native audio',
  'Require generate_audio=true': 'Require generate_audio=true',
  'Supports first frame': 'Supports first frame',
  'Supports last frame': 'Supports last frame',
  'Last frame requires first frame': 'Last frame requires first frame',
  'Reference images cannot be combined with frames':
    'Reference images cannot be combined with frames',
  'Reference audio requires a reference image':
    'Reference audio requires a reference image',
  'Configure at least one ratio': 'Configure at least one ratio',
  'This capability setting is required': 'This capability setting is required',
  'Required native audio must also be supported':
    'Required native audio must also be supported',
  'First frame support is required when the last frame depends on it':
    'First frame support is required when the last frame depends on it',
})
Object.assign(newKeys.zh, {
  megabyai: 'megabyai',
  'Supported aspect ratios': '支持的宽高比',
  'Add an aspect ratio, for example 16:9': '添加宽高比，例如 16:9',
  'Press Enter or comma to add aspect ratios': '按回车或逗号添加宽高比',
  'Supports native audio': '支持原生音频',
  'Require generate_audio=true': '要求 generate_audio=true',
  'Supports first frame': '支持首帧',
  'Supports last frame': '支持尾帧',
  'Last frame requires first frame': '尾帧必须搭配首帧',
  'Reference images cannot be combined with frames':
    '普通参考图不能与首尾帧同时使用',
  'Reference audio requires a reference image': '参考音频必须搭配普通参考图',
  'Configure at least one ratio': '请至少配置一个宽高比',
  'This capability setting is required': '此能力设置为必填项',
  'Required native audio must also be supported':
    '要求原生音频时必须同时启用原生音频支持',
  'First frame support is required when the last frame depends on it':
    '尾帧依赖首帧时必须支持首帧',
})
Object.assign(newKeys['zh-TW'], {
  megabyai: 'megabyai',
  'Supported aspect ratios': '支援的長寬比',
  'Add an aspect ratio, for example 16:9': '新增長寬比，例如 16:9',
  'Press Enter or comma to add aspect ratios': '按 Enter 或逗號新增長寬比',
  'Supports native audio': '支援原生音訊',
  'Require generate_audio=true': '要求 generate_audio=true',
  'Supports first frame': '支援首幀',
  'Supports last frame': '支援尾幀',
  'Last frame requires first frame': '尾幀必須搭配首幀',
  'Reference images cannot be combined with frames':
    '一般參考圖不能與首尾幀同時使用',
  'Reference audio requires a reference image': '參考音訊必須搭配一般參考圖',
  'Configure at least one ratio': '請至少設定一個長寬比',
  'This capability setting is required': '此能力設定為必填項目',
  'Required native audio must also be supported':
    '要求原生音訊時必須同時啟用原生音訊支援',
  'First frame support is required when the last frame depends on it':
    '尾幀依賴首幀時必須支援首幀',
})
Object.assign(newKeys.fr, {
  megabyai: 'megabyai',
  'Supported aspect ratios': "Formats d'image pris en charge",
  'Add an aspect ratio, for example 16:9':
    "Ajouter un format d'image, par exemple 16:9",
  'Press Enter or comma to add aspect ratios':
    "Appuyez sur Entrée ou une virgule pour ajouter des formats d'image",
  'Supports native audio': "Prend en charge l'audio natif",
  'Require generate_audio=true': 'Exiger generate_audio=true',
  'Supports first frame': 'Prend en charge la première image',
  'Supports last frame': 'Prend en charge la dernière image',
  'Last frame requires first frame':
    'La dernière image exige la première image',
  'Reference images cannot be combined with frames':
    'Les images de référence sont incompatibles avec les images de début et de fin',
  'Reference audio requires a reference image':
    'Un audio de référence exige une image de référence',
  'Configure at least one ratio': "Configurez au moins un format d'image",
  'This capability setting is required':
    'Ce paramètre de capacité est obligatoire',
  'Required native audio must also be supported':
    "L'audio natif requis doit également être pris en charge",
  'First frame support is required when the last frame depends on it':
    'La première image doit être prise en charge lorsque la dernière en dépend',
})
Object.assign(newKeys.ja, {
  megabyai: 'megabyai',
  'Supported aspect ratios': '対応アスペクト比',
  'Add an aspect ratio, for example 16:9': 'アスペクト比を追加（例: 16:9）',
  'Press Enter or comma to add aspect ratios':
    'Enter またはカンマでアスペクト比を追加',
  'Supports native audio': 'ネイティブ音声に対応',
  'Require generate_audio=true': 'generate_audio=true を必須にする',
  'Supports first frame': '開始フレームに対応',
  'Supports last frame': '終了フレームに対応',
  'Last frame requires first frame': '終了フレームには開始フレームが必要',
  'Reference images cannot be combined with frames':
    '参照画像と開始・終了フレームは併用不可',
  'Reference audio requires a reference image': '参照音声には参照画像が必要',
  'Configure at least one ratio': 'アスペクト比を1つ以上設定してください',
  'This capability setting is required': 'この機能設定は必須です',
  'Required native audio must also be supported':
    'ネイティブ音声を必須にするには音声対応も有効にしてください',
  'First frame support is required when the last frame depends on it':
    '終了フレームが開始フレームに依存する場合、開始フレームへの対応が必要です',
})
Object.assign(newKeys.ru, {
  megabyai: 'megabyai',
  'Supported aspect ratios': 'Поддерживаемые соотношения сторон',
  'Add an aspect ratio, for example 16:9':
    'Добавьте соотношение сторон, например 16:9',
  'Press Enter or comma to add aspect ratios':
    'Нажмите Enter или запятую, чтобы добавить соотношение сторон',
  'Supports native audio': 'Поддерживает встроенное аудио',
  'Require generate_audio=true': 'Требовать generate_audio=true',
  'Supports first frame': 'Поддерживает первый кадр',
  'Supports last frame': 'Поддерживает последний кадр',
  'Last frame requires first frame':
    'Для последнего кадра требуется первый кадр',
  'Reference images cannot be combined with frames':
    'Референсные изображения нельзя сочетать с первым и последним кадрами',
  'Reference audio requires a reference image':
    'Для референсного аудио требуется референсное изображение',
  'Configure at least one ratio': 'Настройте хотя бы одно соотношение сторон',
  'This capability setting is required':
    'Этот параметр возможностей обязателен',
  'Required native audio must also be supported':
    'Обязательное встроенное аудио должно также поддерживаться',
  'First frame support is required when the last frame depends on it':
    'Если последний кадр зависит от первого, первый кадр должен поддерживаться',
})
Object.assign(newKeys.vi, {
  megabyai: 'megabyai',
  'Supported aspect ratios': 'Tỷ lệ khung hình được hỗ trợ',
  'Add an aspect ratio, for example 16:9': 'Thêm tỷ lệ khung hình, ví dụ 16:9',
  'Press Enter or comma to add aspect ratios':
    'Nhấn Enter hoặc dấu phẩy để thêm tỷ lệ khung hình',
  'Supports native audio': 'Hỗ trợ âm thanh gốc',
  'Require generate_audio=true': 'Yêu cầu generate_audio=true',
  'Supports first frame': 'Hỗ trợ khung hình đầu',
  'Supports last frame': 'Hỗ trợ khung hình cuối',
  'Last frame requires first frame': 'Khung hình cuối yêu cầu khung hình đầu',
  'Reference images cannot be combined with frames':
    'Không thể kết hợp ảnh tham chiếu với khung hình đầu hoặc cuối',
  'Reference audio requires a reference image':
    'Âm thanh tham chiếu yêu cầu một ảnh tham chiếu',
  'Configure at least one ratio': 'Cấu hình ít nhất một tỷ lệ khung hình',
  'This capability setting is required': 'Thiết lập khả năng này là bắt buộc',
  'Required native audio must also be supported':
    'Âm thanh gốc bắt buộc cũng phải được hỗ trợ',
  'First frame support is required when the last frame depends on it':
    'Phải hỗ trợ khung hình đầu khi khung hình cuối phụ thuộc vào nó',
})

for (const locale of ['en', 'zh', 'zh-TW', 'fr', 'ja', 'ru', 'vi']) {
  Object.assign(newKeys[locale], {
    globalaiopc: 'globalaiopc',
  })
}

Object.assign(newKeys.en, {
  'Add a parameter name': 'Add a parameter name',
  'Advanced upstream mapping': 'Advanced upstream mapping',
  'Apply templates for all matching channel models':
    'Apply templates for all matching channel models',
  'Capability template': 'Capability template',
  'Copy a similar template': 'Copy a similar template',
  'Custom template': 'Custom template',
  'Fixed upstream parameters': 'Fixed upstream parameters',
  'No exact template match': 'No exact template match',
  'Official document preset': 'Official document preset',
  'Parameters omitted upstream': 'Parameters omitted upstream',
  'Save current capability as template': 'Save current capability as template',
  'Select an upstream model and apply its capability template':
    'Select an upstream model and apply its capability template',
  'Source document': 'Source document',
  'Upstream value for resolution': 'Upstream value for resolution',
  'Video capability template saved': 'Video capability template saved',
  'Reference images': 'Reference images',
  'Reference videos': 'Reference videos',
  'Reference audios': 'Reference audios',
  Minimum: 'Minimum',
  Maximum: 'Maximum',
  'Aspect ratio is required': 'Aspect ratio is required',
  'Supports duration': 'Supports duration',
  'Duration is required': 'Duration is required',
  'First frame is required': 'First frame is required',
  'Last frame is required': 'Last frame is required',
  'Reference audio requires a visual reference':
    'Reference audio requires a visual reference',
  'Reference media cannot be combined with frames':
    'Reference media cannot be combined with frames',
  'Supports seed': 'Supports seed',
  'Supports watermark': 'Supports watermark',
  'Minimum seed': 'Minimum seed',
  'Maximum seed': 'Maximum seed',
  'Automatically set reference_mode': 'Automatically set reference_mode',
  'Send frames as reference images': 'Send frames as reference images',
  'Reference mode for media': 'Reference mode for media',
  'Reference mode for frames': 'Reference mode for frames',
})

Object.assign(newKeys.zh, {
  'Add a parameter name': '添加参数名',
  'Advanced upstream mapping': '高级上游映射',
  'Apply templates for all matching channel models':
    '为所有匹配的渠道模型应用模板',
  'Capability template': '能力模板',
  'Copy a similar template': '复制相似模板',
  'Custom template': '自定义模板',
  'Fixed upstream parameters': '固定上游参数',
  'No exact template match': '没有精确匹配的模板',
  'Official document preset': '官方文档预设',
  'Parameters omitted upstream': '向上游省略的参数',
  'Save current capability as template': '将当前能力保存为模板',
  'Select an upstream model and apply its capability template':
    '选择上游模型并应用对应的能力模板',
  'Source document': '来源文档',
  'Upstream value for resolution': '分辨率对应的上游值',
  'Video capability template saved': '视频能力模板已保存',
  'Reference images': '参考图片',
  'Reference videos': '参考视频',
  'Reference audios': '参考音频',
  Minimum: '最小',
  Maximum: '最大',
  'Aspect ratio is required': '宽高比必填',
  'Supports duration': '支持时长',
  'Duration is required': '时长必填',
  'First frame is required': '首帧必填',
  'Last frame is required': '尾帧必填',
  'Reference audio requires a visual reference': '参考音频需要视觉参考素材',
  'Reference media cannot be combined with frames': '参考素材不能与首尾帧组合',
  'Supports seed': '支持随机种子',
  'Supports watermark': '支持水印',
  'Minimum seed': '最小随机种子',
  'Maximum seed': '最大随机种子',
  'Automatically set reference_mode': '自动设置 reference_mode',
  'Send frames as reference images': '将首尾帧作为参考图片发送',
  'Reference mode for media': '参考素材模式',
  'Reference mode for frames': '首尾帧模式',
})

Object.assign(newKeys['zh-TW'], {
  'Add a parameter name': '新增參數名稱',
  'Advanced upstream mapping': '進階上游對應',
  'Apply templates for all matching channel models':
    '為所有相符的渠道模型套用範本',
  'Capability template': '能力範本',
  'Copy a similar template': '複製相似範本',
  'Custom template': '自訂範本',
  'Fixed upstream parameters': '固定上游參數',
  'No exact template match': '沒有完全相符的範本',
  'Official document preset': '官方文件預設',
  'Parameters omitted upstream': '向上游省略的參數',
  'Save current capability as template': '將目前能力儲存為範本',
  'Select an upstream model and apply its capability template':
    '選擇上游模型並套用對應的能力範本',
  'Source document': '來源文件',
  'Upstream value for resolution': '解析度對應的上游值',
  'Video capability template saved': '影片能力範本已儲存',
  'Reference images': '參考圖片',
  'Reference videos': '參考影片',
  'Reference audios': '參考音訊',
  Minimum: '最小',
  Maximum: '最大',
  'Aspect ratio is required': '長寬比必填',
  'Supports duration': '支援時長',
  'Duration is required': '時長必填',
  'First frame is required': '首幀必填',
  'Last frame is required': '尾幀必填',
  'Reference audio requires a visual reference': '參考音訊需要視覺參考素材',
  'Reference media cannot be combined with frames': '參考素材不能與首尾幀組合',
  'Supports seed': '支援隨機種子',
  'Supports watermark': '支援浮水印',
  'Minimum seed': '最小隨機種子',
  'Maximum seed': '最大隨機種子',
  'Automatically set reference_mode': '自動設定 reference_mode',
  'Send frames as reference images': '將首尾幀作為參考圖片傳送',
  'Reference mode for media': '參考素材模式',
  'Reference mode for frames': '首尾幀模式',
})

Object.assign(newKeys.fr, {
  'Add a parameter name': 'Ajouter un nom de paramètre',
  'Advanced upstream mapping': 'Mappage amont avancé',
  'Apply templates for all matching channel models':
    'Appliquer les modèles à tous les modèles de canal correspondants',
  'Capability template': 'Modèle de capacités',
  'Copy a similar template': 'Copier un modèle similaire',
  'Custom template': 'Modèle personnalisé',
  'Fixed upstream parameters': 'Paramètres amont fixes',
  'No exact template match': 'Aucun modèle exact',
  'Official document preset': 'Préréglage de la documentation officielle',
  'Parameters omitted upstream': 'Paramètres omis en amont',
  'Save current capability as template':
    'Enregistrer les capacités comme modèle',
  'Select an upstream model and apply its capability template':
    'Sélectionnez un modèle amont et appliquez son modèle de capacités',
  'Source document': 'Document source',
  'Upstream value for resolution': 'Valeur amont de la résolution',
  'Video capability template saved': 'Modèle de capacités vidéo enregistré',
  'Reference images': 'Images de référence',
  'Reference videos': 'Vidéos de référence',
  'Reference audios': 'Audios de référence',
  Minimum: 'Minimum',
  Maximum: 'Maximum',
  'Aspect ratio is required': "Le format d'image est requis",
  'Supports duration': 'Prend en charge la durée',
  'Duration is required': 'La durée est requise',
  'First frame is required': 'La première image est requise',
  'Last frame is required': 'La dernière image est requise',
  'Reference audio requires a visual reference':
    "L'audio de référence nécessite une référence visuelle",
  'Reference media cannot be combined with frames':
    'Les médias de référence ne peuvent pas être combinés avec les images',
  'Supports seed': 'Prend en charge la graine',
  'Supports watermark': 'Prend en charge le filigrane',
  'Minimum seed': 'Graine minimale',
  'Maximum seed': 'Graine maximale',
  'Automatically set reference_mode': 'Définir automatiquement reference_mode',
  'Send frames as reference images': 'Envoyer les images comme références',
  'Reference mode for media': 'Mode de référence des médias',
  'Reference mode for frames': 'Mode de référence des images',
})

Object.assign(newKeys.ja, {
  'Add a parameter name': 'パラメータ名を追加',
  'Advanced upstream mapping': '高度なアップストリームマッピング',
  'Apply templates for all matching channel models':
    '一致する全チャネルモデルにテンプレートを適用',
  'Capability template': '機能テンプレート',
  'Copy a similar template': '類似テンプレートをコピー',
  'Custom template': 'カスタムテンプレート',
  'Fixed upstream parameters': '固定アップストリームパラメータ',
  'No exact template match': '完全一致するテンプレートはありません',
  'Official document preset': '公式ドキュメントのプリセット',
  'Parameters omitted upstream': 'アップストリームで省略するパラメータ',
  'Save current capability as template': '現在の機能をテンプレートとして保存',
  'Select an upstream model and apply its capability template':
    'アップストリームモデルを選択し機能テンプレートを適用します',
  'Source document': '参照ドキュメント',
  'Upstream value for resolution': '解像度のアップストリーム値',
  'Video capability template saved': '動画機能テンプレートを保存しました',
  'Reference images': '参照画像',
  'Reference videos': '参照動画',
  'Reference audios': '参照音声',
  Minimum: '最小',
  Maximum: '最大',
  'Aspect ratio is required': 'アスペクト比は必須です',
  'Supports duration': '時間指定に対応',
  'Duration is required': '時間指定は必須です',
  'First frame is required': '開始フレームは必須です',
  'Last frame is required': '終了フレームは必須です',
  'Reference audio requires a visual reference':
    '参照音声には視覚参照が必要です',
  'Reference media cannot be combined with frames':
    '参照メディアと開始・終了フレームは併用できません',
  'Supports seed': 'シードに対応',
  'Supports watermark': 'ウォーターマークに対応',
  'Minimum seed': '最小シード',
  'Maximum seed': '最大シード',
  'Automatically set reference_mode': 'reference_mode を自動設定',
  'Send frames as reference images': 'フレームを参照画像として送信',
  'Reference mode for media': 'メディアの参照モード',
  'Reference mode for frames': 'フレームの参照モード',
})

Object.assign(newKeys.ru, {
  'Add a parameter name': 'Добавить имя параметра',
  'Advanced upstream mapping': 'Расширенное сопоставление с провайдером',
  'Apply templates for all matching channel models':
    'Применить шаблоны ко всем совпадающим моделям канала',
  'Capability template': 'Шаблон возможностей',
  'Copy a similar template': 'Копировать похожий шаблон',
  'Custom template': 'Пользовательский шаблон',
  'Fixed upstream parameters': 'Фиксированные параметры провайдера',
  'No exact template match': 'Точного шаблона нет',
  'Official document preset': 'Предустановка из официальной документации',
  'Parameters omitted upstream': 'Параметры, не передаваемые провайдеру',
  'Save current capability as template': 'Сохранить возможности как шаблон',
  'Select an upstream model and apply its capability template':
    'Выберите модель провайдера и примените шаблон возможностей',
  'Source document': 'Исходная документация',
  'Upstream value for resolution': 'Значение разрешения у провайдера',
  'Video capability template saved': 'Шаблон возможностей видео сохранён',
  'Reference images': 'Референсные изображения',
  'Reference videos': 'Референсные видео',
  'Reference audios': 'Референсные аудио',
  Minimum: 'Минимум',
  Maximum: 'Максимум',
  'Aspect ratio is required': 'Соотношение сторон обязательно',
  'Supports duration': 'Поддерживает длительность',
  'Duration is required': 'Длительность обязательна',
  'First frame is required': 'Первый кадр обязателен',
  'Last frame is required': 'Последний кадр обязателен',
  'Reference audio requires a visual reference':
    'Референсное аудио требует визуального референса',
  'Reference media cannot be combined with frames':
    'Референсные материалы нельзя сочетать с кадрами',
  'Supports seed': 'Поддерживает seed',
  'Supports watermark': 'Поддерживает водяной знак',
  'Minimum seed': 'Минимальный seed',
  'Maximum seed': 'Максимальный seed',
  'Automatically set reference_mode': 'Автоматически задавать reference_mode',
  'Send frames as reference images':
    'Передавать кадры как референсные изображения',
  'Reference mode for media': 'Режим референсных материалов',
  'Reference mode for frames': 'Режим референсных кадров',
})

Object.assign(newKeys.vi, {
  'Add a parameter name': 'Thêm tên tham số',
  'Advanced upstream mapping': 'Ánh xạ nhà cung cấp nâng cao',
  'Apply templates for all matching channel models':
    'Áp dụng mẫu cho mọi mô hình kênh phù hợp',
  'Capability template': 'Mẫu khả năng',
  'Copy a similar template': 'Sao chép mẫu tương tự',
  'Custom template': 'Mẫu tùy chỉnh',
  'Fixed upstream parameters': 'Tham số nhà cung cấp cố định',
  'No exact template match': 'Không có mẫu khớp chính xác',
  'Official document preset': 'Cấu hình sẵn từ tài liệu chính thức',
  'Parameters omitted upstream': 'Tham số bỏ qua khi gửi lên nhà cung cấp',
  'Save current capability as template': 'Lưu khả năng hiện tại làm mẫu',
  'Select an upstream model and apply its capability template':
    'Chọn mô hình nhà cung cấp và áp dụng mẫu khả năng',
  'Source document': 'Tài liệu nguồn',
  'Upstream value for resolution': 'Giá trị độ phân giải của nhà cung cấp',
  'Video capability template saved': 'Đã lưu mẫu khả năng video',
  'Reference images': 'Ảnh tham chiếu',
  'Reference videos': 'Video tham chiếu',
  'Reference audios': 'Âm thanh tham chiếu',
  Minimum: 'Tối thiểu',
  Maximum: 'Tối đa',
  'Aspect ratio is required': 'Bắt buộc tỷ lệ khung hình',
  'Supports duration': 'Hỗ trợ thời lượng',
  'Duration is required': 'Bắt buộc thời lượng',
  'First frame is required': 'Bắt buộc khung hình đầu',
  'Last frame is required': 'Bắt buộc khung hình cuối',
  'Reference audio requires a visual reference':
    'Âm thanh tham chiếu cần tham chiếu hình ảnh',
  'Reference media cannot be combined with frames':
    'Không thể kết hợp nội dung tham chiếu với khung hình đầu/cuối',
  'Supports seed': 'Hỗ trợ seed',
  'Supports watermark': 'Hỗ trợ hình mờ',
  'Minimum seed': 'Seed tối thiểu',
  'Maximum seed': 'Seed tối đa',
  'Automatically set reference_mode': 'Tự động đặt reference_mode',
  'Send frames as reference images': 'Gửi khung hình dưới dạng ảnh tham chiếu',
  'Reference mode for media': 'Chế độ tham chiếu nội dung',
  'Reference mode for frames': 'Chế độ tham chiếu khung hình',
})

Object.assign(newKeys.en, {
  'All video protocols': 'All video protocols',
  'Apply a capability template': 'Apply a capability template',
  'Built-in template': 'Built-in template',
  'Capability summary': 'Capability summary',
  'Copy video capability template': 'Copy video capability template',
  'Create template': 'Create template',
  'Create video capability template': 'Create video capability template',
  'Delete video capability template': 'Delete video capability template',
  'Edit video capability template': 'Edit video capability template',
  'No templates found': 'No templates found',
  'Search template name or model ID': 'Search template name or model ID',
  'Source URL must be a valid HTTP or HTTPS URL':
    'Source URL must be a valid HTTP or HTTPS URL',
  'Template name': 'Template name',
  'Template source': 'Template source',
  'Templates provide reusable defaults for channel video model capabilities.':
    'Templates provide reusable defaults for channel video model capabilities.',
  'Updated time': 'Updated time',
  'Video capability template deleted': 'Video capability template deleted',
  'Video capability templates': 'Video capability templates',
  'View video capability template': 'View video capability template',
  'Are you sure you want to delete template "{{name}}"? This action cannot be undone.':
    'Are you sure you want to delete template "{{name}}"? This action cannot be undone.',
})

Object.assign(newKeys.zh, {
  'All video protocols': '全部视频协议',
  'Apply a capability template': '套用能力模板',
  'Built-in template': '内置模板',
  'Capability summary': '能力摘要',
  'Copy video capability template': '复制视频能力模板',
  'Create template': '新建模板',
  'Create video capability template': '新建视频能力模板',
  'Delete video capability template': '删除视频能力模板',
  'Edit video capability template': '编辑视频能力模板',
  'No templates found': '未找到模板',
  'Search template name or model ID': '搜索模板名称或模型 ID',
  'Source URL must be a valid HTTP or HTTPS URL':
    '来源 URL 必须是有效的 HTTP 或 HTTPS 地址',
  'Template name': '模板名称',
  'Template source': '模板来源',
  'Templates provide reusable defaults for channel video model capabilities.':
    '模板可作为渠道视频模型能力的可复用默认配置。',
  'Updated time': '更新时间',
  'Video capability template deleted': '视频能力模板已删除',
  'Video capability templates': '视频能力模板',
  'View video capability template': '查看视频能力模板',
  'Are you sure you want to delete template "{{name}}"? This action cannot be undone.':
    '确定要删除模板“{{name}}”吗？此操作无法撤销。',
})

Object.assign(newKeys['zh-TW'], {
  'All video protocols': '全部影片協議',
  'Apply a capability template': '套用能力範本',
  'Built-in template': '內建範本',
  'Capability summary': '能力摘要',
  'Copy video capability template': '複製影片能力範本',
  'Create template': '新增範本',
  'Create video capability template': '新增影片能力範本',
  'Delete video capability template': '刪除影片能力範本',
  'Edit video capability template': '編輯影片能力範本',
  'No templates found': '找不到範本',
  'Search template name or model ID': '搜尋範本名稱或模型 ID',
  'Source URL must be a valid HTTP or HTTPS URL':
    '來源 URL 必須是有效的 HTTP 或 HTTPS 位址',
  'Template name': '範本名稱',
  'Template source': '範本來源',
  'Templates provide reusable defaults for channel video model capabilities.':
    '範本可作為渠道影片模型能力的可重複使用預設設定。',
  'Updated time': '更新時間',
  'Video capability template deleted': '影片能力範本已刪除',
  'Video capability templates': '影片能力範本',
  'View video capability template': '檢視影片能力範本',
  'Are you sure you want to delete template "{{name}}"? This action cannot be undone.':
    '確定要刪除範本「{{name}}」嗎？此操作無法復原。',
})

Object.assign(newKeys.fr, {
  'All video protocols': 'Tous les protocoles vidéo',
  'Apply a capability template': 'Appliquer un modèle de capacités',
  'Built-in template': 'Modèle intégré',
  'Capability summary': 'Résumé des capacités',
  'Copy video capability template': 'Copier le modèle de capacités vidéo',
  'Create template': 'Créer un modèle',
  'Create video capability template': 'Créer un modèle de capacités vidéo',
  'Delete video capability template': 'Supprimer le modèle de capacités vidéo',
  'Edit video capability template': 'Modifier le modèle de capacités vidéo',
  'No templates found': 'Aucun modèle trouvé',
  'Search template name or model ID': 'Rechercher par nom ou ID de modèle',
  'Source URL must be a valid HTTP or HTTPS URL':
    'L’URL source doit être une URL HTTP ou HTTPS valide',
  'Template name': 'Nom du modèle',
  'Template source': 'Source du modèle',
  'Templates provide reusable defaults for channel video model capabilities.':
    'Les modèles fournissent des valeurs réutilisables pour les capacités vidéo des canaux.',
  'Updated time': 'Date de mise à jour',
  'Video capability template deleted': 'Modèle de capacités vidéo supprimé',
  'Video capability templates': 'Modèles de capacités vidéo',
  'View video capability template': 'Afficher le modèle de capacités vidéo',
  'Are you sure you want to delete template "{{name}}"? This action cannot be undone.':
    'Voulez-vous vraiment supprimer le modèle « {{name}} » ? Cette action est irréversible.',
})

Object.assign(newKeys.ja, {
  'All video protocols': 'すべての動画プロトコル',
  'Apply a capability template': '機能テンプレートを適用',
  'Built-in template': '組み込みテンプレート',
  'Capability summary': '機能の概要',
  'Copy video capability template': '動画機能テンプレートをコピー',
  'Create template': 'テンプレートを作成',
  'Create video capability template': '動画機能テンプレートを作成',
  'Delete video capability template': '動画機能テンプレートを削除',
  'Edit video capability template': '動画機能テンプレートを編集',
  'No templates found': 'テンプレートが見つかりません',
  'Search template name or model ID': 'テンプレート名またはモデル ID を検索',
  'Source URL must be a valid HTTP or HTTPS URL':
    '参照 URL は有効な HTTP または HTTPS URL である必要があります',
  'Template name': 'テンプレート名',
  'Template source': 'テンプレートの出典',
  'Templates provide reusable defaults for channel video model capabilities.':
    'テンプレートはチャネルの動画モデル機能に再利用可能な既定値を提供します。',
  'Updated time': '更新日時',
  'Video capability template deleted': '動画機能テンプレートを削除しました',
  'Video capability templates': '動画機能テンプレート',
  'View video capability template': '動画機能テンプレートを表示',
  'Are you sure you want to delete template "{{name}}"? This action cannot be undone.':
    'テンプレート「{{name}}」を削除しますか？この操作は元に戻せません。',
})

Object.assign(newKeys.ru, {
  'All video protocols': 'Все видеопротоколы',
  'Apply a capability template': 'Применить шаблон возможностей',
  'Built-in template': 'Встроенный шаблон',
  'Capability summary': 'Сводка возможностей',
  'Copy video capability template': 'Копировать шаблон возможностей видео',
  'Create template': 'Создать шаблон',
  'Create video capability template': 'Создать шаблон возможностей видео',
  'Delete video capability template': 'Удалить шаблон возможностей видео',
  'Edit video capability template': 'Изменить шаблон возможностей видео',
  'No templates found': 'Шаблоны не найдены',
  'Search template name or model ID': 'Поиск по имени шаблона или ID модели',
  'Source URL must be a valid HTTP or HTTPS URL':
    'URL источника должен быть корректным HTTP- или HTTPS-адресом',
  'Template name': 'Имя шаблона',
  'Template source': 'Источник шаблона',
  'Templates provide reusable defaults for channel video model capabilities.':
    'Шаблоны задают повторно используемые настройки возможностей видеомоделей канала.',
  'Updated time': 'Время обновления',
  'Video capability template deleted': 'Шаблон возможностей видео удалён',
  'Video capability templates': 'Шаблоны возможностей видео',
  'View video capability template': 'Просмотреть шаблон возможностей видео',
  'Are you sure you want to delete template "{{name}}"? This action cannot be undone.':
    'Удалить шаблон «{{name}}»? Это действие нельзя отменить.',
})

Object.assign(newKeys.vi, {
  'All video protocols': 'Tất cả giao thức video',
  'Apply a capability template': 'Áp dụng mẫu khả năng',
  'Built-in template': 'Mẫu tích hợp',
  'Capability summary': 'Tóm tắt khả năng',
  'Copy video capability template': 'Sao chép mẫu khả năng video',
  'Create template': 'Tạo mẫu',
  'Create video capability template': 'Tạo mẫu khả năng video',
  'Delete video capability template': 'Xóa mẫu khả năng video',
  'Edit video capability template': 'Chỉnh sửa mẫu khả năng video',
  'No templates found': 'Không tìm thấy mẫu',
  'Search template name or model ID': 'Tìm theo tên mẫu hoặc ID mô hình',
  'Source URL must be a valid HTTP or HTTPS URL':
    'URL nguồn phải là địa chỉ HTTP hoặc HTTPS hợp lệ',
  'Template name': 'Tên mẫu',
  'Template source': 'Nguồn mẫu',
  'Templates provide reusable defaults for channel video model capabilities.':
    'Mẫu cung cấp cấu hình mặc định có thể tái sử dụng cho khả năng mô hình video của kênh.',
  'Updated time': 'Thời gian cập nhật',
  'Video capability template deleted': 'Đã xóa mẫu khả năng video',
  'Video capability templates': 'Mẫu khả năng video',
  'View video capability template': 'Xem mẫu khả năng video',
  'Are you sure you want to delete template "{{name}}"? This action cannot be undone.':
    'Bạn có chắc muốn xóa mẫu “{{name}}” không? Không thể hoàn tác thao tác này.',
})

Object.assign(newKeys.en, {
  'Platform Task Data': 'Platform Task Data',
  'Request Snapshot': 'Request Snapshot',
  'Task Result': 'Task Result',
})
Object.assign(newKeys.zh, {
  'Platform Task Data': '平台任务数据',
  'Request Snapshot': '请求快照',
  'Task Result': '任务结果',
})
Object.assign(newKeys['zh-TW'], {
  'Platform Task Data': '平台任務資料',
  'Request Snapshot': '請求快照',
  'Task Result': '任務結果',
})
Object.assign(newKeys.fr, {
  'Platform Task Data': 'Données de la tâche de la plateforme',
  'Request Snapshot': 'Instantané de la requête',
  'Task Result': 'Résultat de la tâche',
})
Object.assign(newKeys.ja, {
  'Platform Task Data': 'プラットフォームのタスクデータ',
  'Request Snapshot': 'リクエストスナップショット',
  'Task Result': 'タスク結果',
})
Object.assign(newKeys.ru, {
  'Platform Task Data': 'Данные задачи платформы',
  'Request Snapshot': 'Снимок запроса',
  'Task Result': 'Результат задачи',
})
Object.assign(newKeys.vi, {
  'Platform Task Data': 'Dữ liệu tác vụ nền tảng',
  'Request Snapshot': 'Bản chụp yêu cầu',
  'Task Result': 'Kết quả tác vụ',
})

Object.assign(newKeys.en, {
  'Material and scenario guidelines': 'Material and scenario guidelines',
  'Material and scenario guidelines cannot exceed 20000 characters':
    'Material and scenario guidelines cannot exceed 20000 characters',
  'Describe input material requirements, supported scenarios, incompatible combinations, and known limitations. Markdown is supported.':
    'Describe input material requirements, supported scenarios, incompatible combinations, and known limitations. Markdown is supported.',
  'This customer-facing content appears in the model details page. Markdown is supported.':
    'This customer-facing content appears in the model details page. Markdown is supported.',
})
Object.assign(newKeys.zh, {
  'Material and scenario guidelines': '素材与场景限制',
  'Material and scenario guidelines cannot exceed 20000 characters':
    '素材与场景限制不能超过 20000 个字符',
  'Describe input material requirements, supported scenarios, incompatible combinations, and known limitations. Markdown is supported.':
    '请填写输入素材要求、适用场景、不兼容的组合及已知限制，支持 Markdown。',
  'This customer-facing content appears in the model details page. Markdown is supported.':
    '此内容面向客户，并展示在模型详情页中，支持 Markdown。',
})
Object.assign(newKeys['zh-TW'], {
  'Material and scenario guidelines': '素材與場景限制',
  'Material and scenario guidelines cannot exceed 20000 characters':
    '素材與場景限制不能超過 20000 個字元',
  'Describe input material requirements, supported scenarios, incompatible combinations, and known limitations. Markdown is supported.':
    '請填寫輸入素材要求、適用場景、不相容的組合及已知限制，支援 Markdown。',
  'This customer-facing content appears in the model details page. Markdown is supported.':
    '此內容面向客戶，並顯示於模型詳情頁中，支援 Markdown。',
})
Object.assign(newKeys.fr, {
  'Material and scenario guidelines':
    'Consignes relatives aux ressources et aux scénarios',
  'Material and scenario guidelines cannot exceed 20000 characters':
    'Les consignes relatives aux ressources et aux scénarios ne peuvent pas dépasser 20 000 caractères',
  'Describe input material requirements, supported scenarios, incompatible combinations, and known limitations. Markdown is supported.':
    'Décrivez les exigences relatives aux ressources, les scénarios pris en charge, les combinaisons incompatibles et les limitations connues. Markdown est pris en charge.',
  'This customer-facing content appears in the model details page. Markdown is supported.':
    'Ce contenu destiné aux clients apparaît sur la page de détails du modèle. Markdown est pris en charge.',
})
Object.assign(newKeys.ja, {
  'Material and scenario guidelines': '素材と利用シーンの制限',
  'Material and scenario guidelines cannot exceed 20000 characters':
    '素材と利用シーンの制限は 20000 文字以内で入力してください',
  'Describe input material requirements, supported scenarios, incompatible combinations, and known limitations. Markdown is supported.':
    '入力素材の要件、対応する利用シーン、併用できない組み合わせ、既知の制限を記載してください。Markdown に対応しています。',
  'This customer-facing content appears in the model details page. Markdown is supported.':
    'この顧客向けコンテンツはモデル詳細ページに表示されます。Markdown に対応しています。',
})
Object.assign(newKeys.ru, {
  'Material and scenario guidelines': 'Ограничения материалов и сценариев',
  'Material and scenario guidelines cannot exceed 20000 characters':
    'Ограничения материалов и сценариев не могут превышать 20 000 символов',
  'Describe input material requirements, supported scenarios, incompatible combinations, and known limitations. Markdown is supported.':
    'Опишите требования к входным материалам, поддерживаемые сценарии, несовместимые сочетания и известные ограничения. Поддерживается Markdown.',
  'This customer-facing content appears in the model details page. Markdown is supported.':
    'Этот текст для клиентов отображается на странице сведений о модели. Поддерживается Markdown.',
})
Object.assign(newKeys.vi, {
  'Material and scenario guidelines':
    'Giới hạn về dữ liệu đầu vào và tình huống sử dụng',
  'Material and scenario guidelines cannot exceed 20000 characters':
    'Giới hạn về dữ liệu đầu vào và tình huống sử dụng không được vượt quá 20.000 ký tự',
  'Describe input material requirements, supported scenarios, incompatible combinations, and known limitations. Markdown is supported.':
    'Mô tả yêu cầu về dữ liệu đầu vào, tình huống được hỗ trợ, các tổ hợp không tương thích và giới hạn đã biết. Có hỗ trợ Markdown.',
  'This customer-facing content appears in the model details page. Markdown is supported.':
    'Nội dung dành cho khách hàng này xuất hiện trên trang chi tiết mô hình. Có hỗ trợ Markdown.',
})

for (const [locale, values] of Object.entries({
  en: {
    'Per 1M total tokens': 'Per 1M total tokens',
    'Total token price': 'Total token price',
    'Reserve price per second': 'Reserve price per second',
    'total tokens': 'total tokens',
    Reserve: 'Reserve',
    'final charge uses total tokens; reserve uses requested duration × reserve price per second × output count':
      'Final charge uses total tokens; reserve uses requested duration × reserve price per second × output count',
  },
  zh: {
    'Per 1M total tokens': '每百万总 Token',
    'Total token price': '总 Token 单价',
    'Reserve price per second': '每秒预扣价格',
    'total tokens': '总 Token',
    Reserve: '预扣',
    'final charge uses total tokens; reserve uses requested duration × reserve price per second × output count':
      '最终费用按总 Token 计算；预扣费用按请求时长 × 每秒预扣价格 × 输出数量计算',
  },
  'zh-TW': {
    'Per 1M total tokens': '每百萬總 Token',
    'Total token price': '總 Token 單價',
    'Reserve price per second': '每秒預扣價格',
    'total tokens': '總 Token',
    Reserve: '預扣',
    'final charge uses total tokens; reserve uses requested duration × reserve price per second × output count':
      '最終費用按總 Token 計算；預扣費用按請求時長 × 每秒預扣價格 × 輸出數量計算',
  },
  fr: {
    'Per 1M total tokens': 'Par million de tokens au total',
    'Total token price': 'Prix total des tokens',
    'Reserve price per second': 'Prix de réserve par seconde',
    'total tokens': 'tokens au total',
    Reserve: 'Réserve',
    'final charge uses total tokens; reserve uses requested duration × reserve price per second × output count':
      'Le montant final utilise le total de tokens ; la réserve utilise la durée demandée × le prix de réserve par seconde × le nombre de sorties',
  },
  ja: {
    'Per 1M total tokens': '合計トークン 100 万個あたり',
    'Total token price': '合計トークン単価',
    'Reserve price per second': '1 秒あたりの仮押さえ価格',
    'total tokens': '合計トークン',
    Reserve: '仮押さえ',
    'final charge uses total tokens; reserve uses requested duration × reserve price per second × output count':
      '最終料金は合計トークン数で計算し、仮押さえはリクエスト時間 × 1 秒あたりの仮押さえ価格 × 出力数で計算します',
  },
  ru: {
    'Per 1M total tokens': 'За 1 млн общих токенов',
    'Total token price': 'Цена общих токенов',
    'Reserve price per second': 'Резервная цена за секунду',
    'total tokens': 'общие токены',
    Reserve: 'Резерв',
    'final charge uses total tokens; reserve uses requested duration × reserve price per second × output count':
      'Итоговая сумма рассчитывается по общему числу токенов; резерв — по запрошенной длительности × резервной цене за секунду × числу выходов',
  },
  vi: {
    'Per 1M total tokens': 'Mỗi 1 triệu token tổng',
    'Total token price': 'Đơn giá token tổng',
    'Reserve price per second': 'Giá tạm giữ mỗi giây',
    'total tokens': 'tổng token',
    Reserve: 'Tạm giữ',
    'final charge uses total tokens; reserve uses requested duration × reserve price per second × output count':
      'Chi phí cuối tính theo tổng token; khoản tạm giữ tính theo thời lượng yêu cầu × giá tạm giữ mỗi giây × số đầu ra',
  },
})) {
  Object.assign(newKeys[locale], values)
}

const timeRangeTranslations = {
  en: {
    'Time range': 'Time range',
    'Time zone': 'Time zone',
    'Start time': 'Start time',
    'End time': 'End time',
    'Add time range': 'Add time range',
    'Remove time range': 'Remove time range',
  },
  zh: {
    'Time range': '时间段',
    'Time zone': '时区',
    'Start time': '开始时间',
    'End time': '结束时间',
    'Add time range': '新增时间段',
    'Remove time range': '删除时间段',
  },
  'zh-TW': {
    'Time range': '時間段',
    'Time zone': '時區',
    'Start time': '開始時間',
    'End time': '結束時間',
    'Add time range': '新增時間段',
    'Remove time range': '刪除時間段',
  },
  fr: {
    'Time range': 'Plage horaire',
    'Time zone': 'Fuseau horaire',
    'Start time': 'Heure de début',
    'End time': 'Heure de fin',
    'Add time range': 'Ajouter une plage horaire',
    'Remove time range': 'Supprimer la plage horaire',
  },
  ja: {
    'Time range': '時間帯',
    'Time zone': 'タイムゾーン',
    'Start time': '開始時刻',
    'End time': '終了時刻',
    'Add time range': '時間帯を追加',
    'Remove time range': '時間帯を削除',
  },
  ru: {
    'Time range': 'Временной интервал',
    'Time zone': 'Часовой пояс',
    'Start time': 'Время начала',
    'End time': 'Время окончания',
    'Add time range': 'Добавить интервал',
    'Remove time range': 'Удалить интервал',
  },
  vi: {
    'Time range': 'Khung thời gian',
    'Time zone': 'Múi giờ',
    'Start time': 'Thời gian bắt đầu',
    'End time': 'Thời gian kết thúc',
    'Add time range': 'Thêm khung thời gian',
    'Remove time range': 'Xóa khung thời gian',
  },
}

for (const [locale, translations] of Object.entries(timeRangeTranslations)) {
  Object.assign(newKeys[locale], translations)
}

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
