import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return `${JSON.stringify(obj, null, 2)}\n`
}

const newKeys = {
  en: {
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
