// Qt Widgets + LayerShellQt achievement notification.
// Configure/build from this directory with CMake; see README.md.

#include <QApplication>
#include <QConicalGradient>
#include <QJsonDocument>
#include <QJsonObject>
#include <QWindow>
#include <algorithm>
#include <cstdio>
#include <QDebug>
#include <QEvent>
#include <QFile>
#include <QFrame>
#include <QGraphicsBlurEffect>
#include <QGraphicsDropShadowEffect>
#include <QGridLayout>
#include <QHBoxLayout>
#include <QLabel>
#include <QPainter>
#include <QPainterPath>
#include <QPixmap>
#include <QProgressBar>
#include <QScreen>
#include <QTextLayout>
#include <QTimer>
#include <QVariantAnimation>
#include <QVBoxLayout>
#include <QWidget>

#ifdef SENTINEL_QSS_HOT_RELOAD
#include <QFileInfo>
#include <QFileSystemWatcher>
#endif

#ifdef Q_OS_LINUX
#include <LayerShellQt/Window>
#endif

// Reserve a bounded number of lines and elide the last one at the actual width.
class ClampedLabel : public QLabel {
public:
  ClampedLabel(const QString &text, int maxLines, QWidget *parent)
      : QLabel(text, parent), maxLines_(maxLines) {
    setTextFormat(Qt::PlainText);
    setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Fixed);
  }

  QSize sizeHint() const override {
    return QSize(0, fontMetrics().lineSpacing() * maxLines_);
  }

  QSize minimumSizeHint() const override { return sizeHint(); }

protected:
  void paintEvent(QPaintEvent *) override {
    QPainter painter(this);
    painter.setPen(palette().color(QPalette::WindowText));
    const auto metrics = painter.fontMetrics();
    QString content = text();
    content.replace('\n', ' ');
    content.replace('\r', ' ');
    QTextLayout layout(content, font());
    QTextOption options;
    options.setWrapMode(QTextOption::WrapAtWordBoundaryOrAnywhere);
    layout.setTextOption(options);
    layout.beginLayout();
    for (int i = 0; i < maxLines_; ++i) {
      QTextLine line = layout.createLine();
      if (!line.isValid())
        break;
      line.setLineWidth(width());
      const int y = i * metrics.lineSpacing();
      if (i == maxLines_ - 1) {
        painter.drawText(QPoint(0, y + metrics.ascent()),
                         metrics.elidedText(content.mid(line.textStart()),
                                            Qt::ElideRight, width()));
      } else {
        line.draw(&painter, QPointF(0, y));
      }
    }
    layout.endLayout();
  }

private:
  int maxLines_;
};

// QSS has no CSS filter/keyframe support. Paint and blur a separate layer so
// the artwork and its thin border stay sharp while the gold highlight moves.
class RareAchievementGlow : public QWidget {
public:
  RareAchievementGlow(QWidget *target, QWidget *parent)
      : QWidget(parent), target_(target) {
    setAttribute(Qt::WA_TransparentForMouseEvents);
    target_->installEventFilter(this);
    syncGeometry();
    auto *blur = new QGraphicsBlurEffect(this);
    blur->setBlurRadius(6);
    setGraphicsEffect(blur);

    auto *spin = new QVariantAnimation(this);
    spin->setStartValue(0.0);
    spin->setEndValue(360.0);
    spin->setDuration(3000);
    spin->setLoopCount(-1);
    QObject::connect(spin, &QVariantAnimation::valueChanged, this,
                     [this](const QVariant &value) {
                       angle_ = value.toReal();
                       update();
                     });
    spin->start();
  }

protected:
  bool eventFilter(QObject *object, QEvent *event) override {
    if (object == target_ &&
        (event->type() == QEvent::Move || event->type() == QEvent::Resize))
      syncGeometry();
    return QWidget::eventFilter(object, event);
  }

  void paintEvent(QPaintEvent *) override {
    QPainter painter(this);
    painter.setRenderHint(QPainter::Antialiasing);
    // Qt conical gradients turn counterclockwise; CSS gradients turn clockwise.
    QConicalGradient gradient(QRectF(rect()).center(), 90 - angle_);
    gradient.setColorAt(0, Qt::transparent);
    gradient.setColorAt(60.0 / 360, Qt::transparent);
    gradient.setColorAt(140.0 / 360, QColor(255, 174, 0, 64));
    gradient.setColorAt(200.0 / 360, QColor(255, 244, 178, 242));
    gradient.setColorAt(245.0 / 360, QColor(255, 174, 0, 242));
    gradient.setColorAt(305.0 / 360, QColor(255, 214, 77, 51));
    gradient.setColorAt(1, Qt::transparent);
    // Blur a narrow ring instead of a filled square, so the bright portion
    // doesn't spill out as a large blob when it passes an edge or corner.
    painter.setPen(QPen(QBrush(gradient), 2));
    painter.setBrush(Qt::NoBrush);
    painter.drawRoundedRect(QRectF(rect()).adjusted(3, 3, -3, -3), 11, 11);
  }

private:
  void syncGeometry() {
    const QRect frame(target_->mapTo(parentWidget(), QPoint()), target_->size());
    setGeometry(frame.adjusted(-4, -4, 4, 4));
  }

  QWidget *target_;
  qreal angle_ = 0;
};

struct CustomNotificationData {
  QString title;
  QString description;
  QString gameName;
  QString iconPath;
  qint64 progress = 0;
  qint64 maxProgress = 0;
  bool isProgress = false;
  bool isRare = false;
  int durationMs = 0;
};

static void readCustomNotification(CustomNotificationData &data) {
  QFile input;
  (void)input.open(stdin, QIODevice::ReadOnly);
  const QJsonObject object = QJsonDocument::fromJson(input.readAll()).object();
  data.title = object.value("title").toString();
  data.description = object.value("description").toString();
  data.gameName = object.value("gameName").toString();
  data.iconPath = object.value("iconPath").toString();
  data.progress = object.value("progress").toInteger();
  data.maxProgress = object.value("maxProgress").toInteger();
  data.isProgress = object.value("isProgress").toBool();
  data.isRare = object.value("isRare").toBool();
  data.durationMs = object.value("durationMs").toInt();
}

int main(int argc, char *argv[]) {
#ifdef Q_OS_LINUX
  // Select the Wayland layer-shell plugin before Qt creates any windows.
  qputenv("QT_QPA_PLATFORM", "wayland");
  qputenv("QT_WAYLAND_SHELL_INTEGRATION", "layer-shell");
#endif

  QApplication app(argc, argv);
  app.setApplicationName("sentinel-custom-notification");

  CustomNotificationData data;
  readCustomNotification(data);

  QWidget notification;
  Qt::WindowFlags windowFlags = Qt::FramelessWindowHint | Qt::Tool |
                                Qt::WindowStaysOnTopHint;
#ifdef Q_OS_LINUX
  windowFlags |= Qt::WindowDoesNotAcceptFocus | Qt::WindowTransparentForInput;
#endif

  notification.setWindowFlags(windowFlags);
  notification.setAttribute(Qt::WA_TranslucentBackground);
  notification.setAttribute(Qt::WA_ShowWithoutActivating);
#ifdef Q_OS_MACOS
  // Keep the notification visible when another application is active.
  notification.setAttribute(Qt::WA_MacAlwaysShowToolWindow);
#endif

  notification.setFixedWidth(450);

#ifdef SENTINEL_QSS_HOT_RELOAD
  const QString sourceStylesheetPath =
      QStringLiteral(SENTINEL_CUSTOM_NOTIFICATION_STYLESHEET);
  const bool liveStylesheet = QFile::exists(sourceStylesheetPath);
  const QString stylesheetPath = liveStylesheet
      ? sourceStylesheetPath
      : QStringLiteral(":/sentinel/custom-notification.qss");
  QFileSystemWatcher stylesheetWatcher;
  QTimer stylesheetReloadTimer;
  stylesheetReloadTimer.setSingleShot(true);
  stylesheetReloadTimer.setInterval(100);
#else
  const QString stylesheetPath = QStringLiteral(":/sentinel/custom-notification.qss");
#endif

  auto reloadStylesheet = [&]() {
#ifdef SENTINEL_QSS_HOT_RELOAD
    // Editors may replace the file on save, removing the original file watch.
    if (liveStylesheet && QFile::exists(stylesheetPath) &&
        !stylesheetWatcher.files().contains(stylesheetPath)) {
      if (!stylesheetWatcher.addPath(stylesheetPath))
        qWarning() << "Cannot watch stylesheet:" << stylesheetPath;
    }
#endif

    QFile stylesheet(stylesheetPath);
    if (!stylesheet.open(QIODevice::ReadOnly)) {
      qWarning() << "Cannot read stylesheet:" << stylesheetPath
                 << stylesheet.errorString();
      return false;
    }

    const QString styles = QString::fromUtf8(stylesheet.readAll());
    if (stylesheet.error() != QFileDevice::NoError) {
      qWarning() << "Cannot read stylesheet:" << stylesheet.errorString();
      return false;
    }

    if (notification.styleSheet() != styles) {
      notification.setStyleSheet(styles);
      qInfo() << "Stylesheet loaded:" << stylesheetPath;
      if (notification.layout()) {
        notification.layout()->activate();
        notification.resize(notification.width(), notification.layout()->totalHeightForWidth(notification.width()));
      }
    }

    return true;
  };

#ifdef SENTINEL_QSS_HOT_RELOAD
  QObject::connect(&stylesheetReloadTimer, &QTimer::timeout, &notification,
                   reloadStylesheet);

  // Watch the directory too so atomic saves and file recreation reload styles.
  if (liveStylesheet && !stylesheetWatcher.addPath(QFileInfo(stylesheetPath).absolutePath()))
    qWarning() << "Cannot watch stylesheet directory:" << stylesheetPath;

  auto scheduleReload = [&](const QString &) {
    stylesheetReloadTimer.start();
  };
  QObject::connect(&stylesheetWatcher, &QFileSystemWatcher::fileChanged,
                   &notification, scheduleReload);
  QObject::connect(&stylesheetWatcher, &QFileSystemWatcher::directoryChanged,
                   &notification, scheduleReload);
#endif

  if (!reloadStylesheet())
    return 3;

  auto *outer = new QVBoxLayout(&notification);
  outer->setContentsMargins(0, 0, 0, 0);
  outer->setSpacing(0);

  auto *card = new QFrame(&notification);
  card->setObjectName("card");
  outer->addWidget(card);

  auto *cardLayout = new QVBoxLayout(card);
  cardLayout->setContentsMargins(16, 16, 16, 16);
  cardLayout->setSpacing(8);

  auto *gameName = new QLabel(data.gameName, card);
  gameName->setObjectName("gameName");
  gameName->setTextFormat(Qt::PlainText);
  gameName->setWordWrap(true);
  cardLayout->addWidget(gameName);

  auto *row = new QHBoxLayout;
  row->setContentsMargins(0, 0, 0, 0);
  row->setSpacing(12);
  cardLayout->addLayout(row);

  // All notification types share the same frame and horizontal position.
  auto *iconContainer = new QWidget(card);
  auto *iconLayers = new QGridLayout(iconContainer);
  iconLayers->setContentsMargins(0, 0, 0, 0);
  iconLayers->setSpacing(0);
  iconLayers->setSizeConstraint(QLayout::SetFixedSize);
  QPixmap artwork(data.iconPath);
  if (artwork.isNull())
    artwork.load(":/sentinel/default-icon.png");
  auto *icon = new QLabel(iconContainer);
  icon->setObjectName("icon");
  icon->setProperty("rare", data.isRare);
  const int iconSize = 100;
  icon->setFixedSize(iconSize, iconSize);
  icon->setAlignment(Qt::AlignCenter);

  if (!artwork.isNull()) {
    const QPixmap scaled =
        artwork.scaled(iconSize - 4, iconSize - 4, Qt::KeepAspectRatio, Qt::SmoothTransformation);
    QPixmap rounded(scaled.size());
    rounded.fill(Qt::transparent);
    QPainter painter(&rounded);
    painter.setRenderHint(QPainter::Antialiasing);
    QPainterPath clip;
    // Match the 10px frame radius inside the artwork's 2px inset.
    clip.addRoundedRect(QRectF(rounded.rect()), 8, 8);
    painter.setClipPath(clip);
    painter.drawPixmap(0, 0, scaled);
    painter.end();
    icon->setPixmap(rounded);
  } else {
    icon->setText("Sentinel");
  }
  if (data.isRare) {
    auto *glow = new QGraphicsDropShadowEffect(icon);
    glow->setColor(QColor(255, 174, 0, 178));
    glow->setBlurRadius(20);
    glow->setOffset(0, 0);
    icon->setGraphicsEffect(glow);
  }
  iconLayers->addWidget(icon, 0, 0, Qt::AlignCenter);
  row->addWidget(iconContainer, 0, Qt::AlignVCenter);
  if (data.isRare) {
    // A sibling paint layer can overflow the frame without adding layout margins.
    auto *animatedGlow = new RareAchievementGlow(iconContainer, card);
    animatedGlow->stackUnder(iconContainer);
  }

  // Match the visible icon, excluding the extra space reserved for its glow.
  auto *content = new QWidget(card);
  if (data.isProgress)
    content->setFixedHeight(iconSize);
  else
    content->setMinimumHeight(iconSize);
  auto *text = new QVBoxLayout(content);
  // Align progress content with the artwork inside the icon's 2px frame inset.
  const int contentInset = data.isProgress ? 2 : 0;
  text->setContentsMargins(0, contentInset, 0, contentInset);
  text->setSpacing(data.isProgress ? 4 : 6);
  if (!data.isProgress)
    text->setAlignment(Qt::AlignVCenter);

  row->addWidget(content, 1, Qt::AlignVCenter);
  auto *title = new ClampedLabel(data.title, 1, content);
  title->setObjectName("title");
  text->addWidget(title);

  auto *description = new ClampedLabel(data.description, 2, content);
  description->setObjectName("description");
  text->addWidget(description);
  if (data.isProgress) {
    text->addStretch();
    auto *progressDetails = new QVBoxLayout;
    progressDetails->setContentsMargins(0, 0, 0, 0);
    progressDetails->setSpacing(2);
    text->addLayout(progressDetails);
    auto *progressCount = new QLabel(
        QString::number(data.progress) + "/" + QString::number(data.maxProgress), content);
    progressCount->setObjectName("progressCount");
    progressCount->setTextFormat(Qt::PlainText);
    progressCount->setAlignment(Qt::AlignRight | Qt::AlignVCenter);
    progressDetails->addWidget(progressCount);

    auto *progress = new QProgressBar(content);
    progress->setRange(0, 1000);
    const double ratio = data.maxProgress > 0
        ? std::clamp(double(data.progress) / double(data.maxProgress), 0.0, 1.0)
        : 0.0;
    progress->setValue(qRound(ratio * 1000));
    progress->setTextVisible(false);
    progress->setFixedHeight(9);
    progressDetails->addWidget(progress);
  }

  notification.ensurePolished();
  outer->activate();
  notification.resize(notification.width(), outer->totalHeightForWidth(notification.width()));

  // Create the native QWindow while hidden, then configure the platform role.
#ifdef Q_OS_LINUX
  (void)notification.winId();
  auto *layer = LayerShellQt::Window::get(notification.windowHandle());
  if (!layer) {
    qCritical() << "Layer-shell window unavailable";
    return 3;
  }
  layer->setScope("sentinel-custom-notification");
  layer->setLayer(LayerShellQt::Window::LayerOverlay);
  layer->setAnchors(
      LayerShellQt::Window::Anchors(LayerShellQt::Window::AnchorBottom) |
      LayerShellQt::Window::AnchorRight);
  layer->setMargins(QMargins(0, 0, 24, 24));
  layer->setExclusiveZone(0);
  layer->setKeyboardInteractivity(
      LayerShellQt::Window::KeyboardInteractivityNone);
  layer->setActivateOnShow(false);
  layer->setWantsToBeOnActiveScreen(true);
#else
  // Position the notification in the lower-right corner on macOS.
  if (auto *screen = QGuiApplication::primaryScreen())
    notification.move(screen->availableGeometry().bottomRight() -
               QPoint(notification.width() + 24, notification.height() + 24));
#endif

  notification.show();
  QTimer::singleShot(data.durationMs, &app, &QApplication::quit);
  return app.exec();
}
