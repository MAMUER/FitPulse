import { useLocation } from 'react-router-dom';
import { useApp } from '../contexts/AppContext';

const PRIVACY = `Политика конфиденциальности — FitPulse

Дата последнего обновления: 2026-07-29

1. Общие положения
FitPulse («мы», «наша», «наш») — это открытая платформа для фитнес- и health-трекинга. Настоящая Политика конфиденциальности описывает, как мы собираем, используем, храним и защищаем ваши данные при использовании сервиса.

2. Какие данные мы собираем
2.1 Данные учётной записи
- Адрес электронной почты (для аутентификации и восстановления доступа)
- Имя пользователя и отображаемое имя
- Хэш пароля (Argon2id, никогда не хранится в открытом виде)

2.2 Данные о здоровье и активности
- Биометрические показатели: пульс, SpO2, шаги, калории, сон
- Информация об устройстве: тип устройства, модель, время синхронизации
- Данные тренировок: упражнения, подходы, повторения, веса

2.3 Данные OAuth (при использовании Yandex ID)
- Идентификатор аккаунта Yandex (yandex_id из userinfo)
- Адрес электронной почты (из Yandex ID)
- Основная информация профиля (имя, URL фото)

2.4 Технические данные
- IP-адрес (для ограничения частоты запросов и безопасности)
- User-Agent (для аналитики)
- Логи API-запросов (структурированные логи с санитизированным вводом)
- Метаданные TLS-подключения (SNI, шифр)

 3. Как мы используем ваши данные
 - Аутентификация: проверка личности, управление сессиями
 - Предоставление сервиса: синхронизация данных между устройствами и сервисами
 - Аналитика: агрегированная статистика без привязки к личности
 - Безопасность: обнаружение и предотвращение нарушений, мошенничества
 - Уведомления: отправка служебных сообщений (по согласию пользователя)
 
 3.1 Специальные категории данных
 - Сведения о здоровье (заболевания, симптомы, аллергии, противопоказания) и менструальном цикле являются специальными категориями персональных данных по GDPR Art. 9 и 152-ФЗ.
 - Обработка этих данных осуществляется только с вашим явным согласием.
 - Вы можете отозвать согласие в любое время, удалив соответствующие данные в разделе «Профиль» или запросив удаление аккаунта.
 
 4. Хранение и сроки хранения
- База данных: PostgreSQL с шифрованием AES-256-GCM (pgsodium)
- Резервные копии: шифруются AES-256-CBC, хранятся 7 дней
- Срок хранения: данные о здоровье хранятся до запроса на удаление аккаунта
- Анонимизация: после удаления аккаунта данные псевдонимизируются в течение 30 дней

5. Защита данных
Мы используем отраслевые стандарты безопасности:
- TLS 1.3 для всех внешних соединений
- Хэширование паролей Argon2id (память 64 MB, 3 итерации, параллелизм 1)
- Шифрование AES-256-GCM для чувствительных медицинских данных
- mTLS для внутренней коммуникации между сервисами
- Ограничение частоты запросов и WAF (ModSecurity + OWASP CRS v4)

6. Передача данных третьим лицам
Мы не продаём и не передаём ваши персональные данные третьим лицам, кроме случаев, предусмотренных законом или технической интеграцией:
- Yandex ID 2.0 — аутентификация
- Open Wearables — агрегированные данные здоровья из различных источников
- Telegram Bot API — уведомления (по согласию)
- DuckDNS — доменная запись

7. Ваши права (GDPR / CCPA)
- Доступ: запрос копии персональных данных
- Исправление: корректировка неточных данных
- Удаление: запрос на удаление аккаунта и данных
- Переносимость: экспорт данных в JSON/CSV
- Возражение: отказ от несущественной обработки данных

Контакт для запросов: mihnikolaenko12@yandex.ru

8. Защита прав детей
Сервис не предназначен для пользователей до 13 лет. Мы не собираем персональные данные о несовершеннолетних.

9. Изменения политики
Мы можем обновлять политику periodically. Существенные изменения будут communicated через email или in-app уведомление.

10. Контакты
По вопросам конфиденциальности: mihnikolaenko12@yandex.ru`;

const TERMS = `Пользовательское соглашение — FitPulse

Дата последнего обновления: 2026-07-29

1. Принятие условий
Используя FitPulse («Сервис»), вы соглашаетесь с настоящими Условиями использования. Если вы не согласны, пожалуйста, не используйте Сервис.

2. Описание сервиса
FitPulse — это открытая платформа для фитнес- и health-трекинга, которая позволяет:
- отслеживать биометрические показатели (пульс, SpO2, шаги, калории);
- синхронизировать данные через Open Wearables агрегатор;
- отслеживать прогресс в тренировках;
- получать аналитику здоровья.

3. Учётные записи
- При создании аккаунта необходимо указывать точные и полные данные
- Вы несёте ответственность за сохранность своих учётных данных
- Немедленно уведомляйте нас о несанкционированном доступе
- Запрещено создание множественных аккаунтов одним лицом

4. Допустимое использование
Вы соглашаетесь не:
- использовать Сервис для незаконных целей;
- пытаться получить несанкционированный доступ к любой части Сервиса;
- вмешиваться в работу Сервиса или нарушать его целостность;
- использовать автоматические средства доступа без нашего разрешения;
- осуществлять реверс-инжиниринг или попытки извлечения исходного кода.

5. Отказ от ответственности за медицинские данные
- FitPulse предоставляет только инструменты трекинга и аналитики;
- Мы не даём медицинских советов, диагнозов или лечения;
- При медицинских решениях всегда консультируйтесь с квалифицированным врачом;
- Точность показателей здоровья зависит от датчиков устройств и данных пользователя.

6. Интеллектуальная собственность
- Исходный код FitPulse доступен под открытой лицензией — см. файл LICENSE в репозитории
- Товарные знаки и логотипы принадлежат соответствующим правообладателям
- Вы сохраняете права на свои данные о здоровье

7. Интеграции с третьими сервисами
Сервис интегрируется с Open Wearables для агрегации данных здоровья из различных источников. Ваше использование этих сервисов регулируется их соответствующими условиями и политиками конфиденциальности.

8. Ограничение ответственности
FitPulse предоставляется «как есть» без гарантий. Мы не несём ответственности за:
- потерю или повреждение данных;
- перерывы в работе сервиса;
- неточность измерения показателей здоровья;
- косвенные или побочные убытки.

9. Прекращение доступа
Мы оставляем за собой право приостановить или удалить аккаунты, нарушающие эти условия, с уведомлением или без него.

10. Применимое право
Условия регулируются действующим законодательством. Споры решаются в соответствии с юрисдикцией оператора сервиса.

11. Контакты
По вопросам, связанным с условиями: mihnikolaenko12@yandex.ru`;

const PRIVACY_EN = `Privacy Policy — FitPulse

Last updated: 2026-07-29

1. General provisions
FitPulse ("we", "our") is an open platform for fitness and health tracking. This Privacy Policy describes how we collect, use, store and protect your data when you use the service.

2. What data we collect
2.1 Account data
- Email address (for authentication and account recovery)
- Username and display name
- Password hash (Argon2id, never stored in plain text)

2.2 Health and activity data
- Biometric metrics: heart rate, SpO2, steps, calories, sleep
- Device information: device type, model, sync time
- Workout data: exercises, sets, reps, weights

2.3 OAuth data (when using Yandex ID)
- Yandex account identifier (yandex_id from userinfo)
- Email address (from Yandex ID)
- Basic profile information (name, photo URL)

2.4 Technical data
- IP address (for rate limiting and security)
- User-Agent (for analytics)
- API request logs (structured logs with sanitized input)
- TLS connection metadata (SNI, cipher)

3. How we use your data
- Authentication: identity verification, session management
- Service delivery: data sync between devices and services
- Analytics: aggregated statistics without personal identification
- Security: detection and prevention of violations and fraud
- Notifications: service messages (with user consent)

3.1 Special categories of data
- Health information (conditions, symptoms, allergies, contraindications) and menstrual cycle data are special categories of personal data under GDPR Art. 9 and 152-FZ.
- Processing of this data is carried out only with your explicit consent.
- You can withdraw consent at any time by deleting the relevant data in the "Profile" section or by requesting account deletion.

4. Storage and retention
- Database: PostgreSQL with AES-256-GCM encryption (pgsodium)
- Backups: encrypted with AES-256-CBC, stored for 7 days
- Retention period: health data is stored until account deletion is requested
- Anonymization: after account deletion, data is pseudonymized within 30 days

5. Data protection
We use industry-standard security measures:
- TLS 1.3 for all external connections
- Argon2id password hashing (64 MB memory, 3 iterations, parallelism 1)
- AES-256-GCM encryption for sensitive medical data
- mTLS for internal communication between services
- Rate limiting and WAF (ModSecurity + OWASP CRS v4)

6. Data sharing with third parties
We do not sell or transfer your personal data to third parties, except as required by law or technical integration:
- Yandex ID 2.0 — authentication
- Open Wearables — aggregated health data from various sources
- Telegram Bot API — notifications (with consent)
- DuckDNS — DNS record

7. Your rights (GDPR / CCPA)
- Access: request a copy of your personal data
- Rectification: correct inaccurate data
- Deletion: request account and data deletion
- Portability: export data in JSON/CSV
- Objection: object to non-essential processing

Contact for requests: mihnikolaenko12@yandex.ru

8. Children's privacy
The service is not intended for users under 13. We do not collect personal data from minors.

9. Policy changes
We may update this policy periodically. Material changes will be communicated via email or in-app notification.

10. Contacts
For privacy questions: mihnikolaenko12@yandex.ru`;

const TERMS_EN = `Terms of Use — FitPulse

Last updated: 2026-07-29

1. Acceptance of terms
By using FitPulse (the "Service"), you agree to these Terms of Use. If you do not agree, please do not use the Service.

2. Service description
FitPulse is an open platform for fitness and health tracking that allows you to:
- track biometric metrics (heart rate, SpO2, steps, calories);
- sync data through the Open Wearables aggregator;
- track workout progress;
- get health analytics.

3. Accounts
- You must provide accurate and complete information when creating an account
- You are responsible for the security of your credentials
- Immediately notify us of unauthorized access
- Creating multiple accounts by one person is prohibited

4. Acceptable use
You agree not to:
- use the Service for illegal purposes;
- attempt to gain unauthorized access to any part of the Service;
- interfere with the operation of the Service or violate its integrity;
- use automated access methods without our permission;
- reverse engineer or attempt to extract source code.

5. Disclaimer regarding medical data
- FitPulse provides only tracking and analytics tools;
- We do not provide medical advice, diagnoses or treatment;
- Always consult a qualified physician for medical decisions;
- The accuracy of health metrics depends on device sensors and user data.

6. Intellectual property
- FitPulse source code is available under an open license — see the LICENSE file in the repository
- Trademarks and logos belong to their respective owners
- You retain rights to your health data

7. Third-party integrations
The Service integrates with Open Wearables to aggregate health data from various sources. Your use of these services is governed by their respective terms and privacy policies.

8. Limitation of liability
FitPulse is provided "as is" without warranties. We are not liable for:
- data loss or corruption;
- service interruptions;
- inaccurate measurement of health metrics;
- indirect or consequential damages.

9. Termination of access
We reserve the right to suspend or delete accounts that violate these terms, with or without notice.

10. Governing law
These terms are governed by applicable law. Disputes are resolved in accordance with the jurisdiction of the service operator.

11. Contacts
For questions regarding the terms: mihnikolaenko12@yandex.ru`;

export default function Legal() {
  const { t, backFromLegal } = useApp();
  const { pathname } = useLocation();

  let kind;
  if (pathname.includes('privacy')) {
    kind = 'privacy';
  } else if (pathname.includes('terms')) {
    kind = 'terms';
  } else {
    kind = 'consent';
  }

  let title;
  if (kind === 'terms') {
    title = t('Пользовательское соглашение', 'Terms of Use');
  } else if (kind === 'consent') {
    title = t(
      'Соглашение об использовании персональных данных',
      'Personal Data Agreement'
    );
  } else {
    title = t('Политика конфиденциальности', 'Privacy Policy');
  }

  let body;
  if (kind === 'terms') {
    body = t(TERMS, TERMS_EN);
  } else if (kind === 'consent') {
    body = t(
      'Согласие на обработку персональных данных.',
      'Personal Data Processing Consent.'
    );
  } else {
    body = t(PRIVACY, PRIVACY_EN);
  }

  return (
    <section className='legal'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{title}</div>
        </div>
        <div className='panel-body'>
          <pre className='legal-body'>{body}</pre>
          <button
            type='button'
            className='secondary full'
            onClick={backFromLegal}
          >
            {t('Назад', 'Back')}
          </button>
        </div>
      </div>
    </section>
  );
}
