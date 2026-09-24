import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { backendRequest } from '../utils/backendRequest';
import {
  formatRelativeDate,
  getDateKey,
  isDateAllowed,
  safeText,
} from '../utils/helpers';
import { getMonthName, t } from '../utils/i18n';

const KEY = 'fitpulse-merged-v9';

const defaultState = {
  screen: 'login',
  theme: 'dark',
  highContrast: false,
  language: 'ru',
  period: 'День',
  monthOpen: false,
  selectedMeal: null,
  selectedMetric: null,
  selectedWorkout: null,
  selectedPlace: 'Дом',
  nutritionView: 'День',
  calendarMonthIndex: new Date().getMonth(),
  calendarYear: new Date().getFullYear(),
  profileImage: '',
  profile: {
    name: 'Михаил',
    gender: 'Мужской',
    email: 'mih@example.com',
    phone: '+7 999 123 45 67',
    twoFactor: false,
    backupCodesRemaining: 0,
    assistantStyle: 'Дружелюбный',
    devices: ['Apple Watch', 'Смарт-весы'],
    bio: 'Люблю спорт и здоровое питание 💪',
    status: 'Активен',
    friends: 12,
    password: '',
  },
  twoFactorSetup: null,
  twoFactorTempToken: null,
  messages: [
    {
      id: 'default-ai',
      type: 'ai',
      text: 'Привет! Я FitPulse AI. Могу помочь с питанием, тренировками, календарём и ответить на вопросы.',
    },
  ],
  events: [
    {
      type: 'training',
      title: 'Силовая база',
      info: 'Ноги, корпус и стабилизация',
      time: '18:30',
      date: '24 июля',
      color: '#6fae20',
    },
    {
      type: 'food',
      title: 'Курица с гречкой',
      info: '480 ккал · Б 38 г · Ж 14 г · У 48 г',
      time: '13:20',
      date: '24 июля',
      color: '#f5d45d',
    },
    {
      type: 'recovery',
      title: 'Восстановление',
      info: 'Дыхание и растяжка',
      time: '21:30',
      date: '25 июля',
      color: '#ff6375',
    },
  ],
  weightHistory: [
    ['01 июля', '76.1 кг'],
    ['08 июля', '75.4 кг'],
    ['15 июля', '74.8 кг'],
    ['22 июля', '74.2 кг'],
  ],
  meals: {
    breakfast: {
      label: 'Завтрак',
      icon: 'fa-sun',
      time: '08:00',
      title: 'Овсянка с ягодами',
      kcal: 360,
      macros: 'Б 16 г · Ж 10 г · У 51 г',
      image:
        'https://images.unsplash.com/photo-1490474418585-ba9bad8fd0ea?w=500',
    },
    lunch: {
      label: 'Обед',
      icon: 'fa-bowl-food',
      time: '13:20',
      title: 'Курица с гречкой',
      kcal: 480,
      macros: 'Б 38 г · Ж 14 г · У 48 г',
      image: 'https://images.unsplash.com/photo-1547592180-85f173990554?w=500',
    },
    dinner: {
      label: 'Ужин',
      icon: 'fa-moon',
      time: '19:30',
      title: 'Боул с лососем',
      kcal: 540,
      macros: 'Б 34 г · Ж 22 г · У 48 г',
      image:
        'https://images.unsplash.com/photo-1512621776951-a57141f2eefd?w=500',
    },
    snack: {
      label: 'Полдник',
      icon: 'fa-apple-whole',
      time: '16:30',
      title: 'Йогурт и ягоды',
      kcal: 210,
      macros: 'Б 18 г · Ж 5 г · У 24 г',
      image:
        'https://images.unsplash.com/photo-1488477181946-6428a0291777?w=500',
    },
  },
  alternatives: {
    breakfast: [
      [
        'Омлет со шпинатом',
        390,
        'Б 27 г · Ж 21 г · У 18 г',
        'https://images.unsplash.com/photo-1525351484163-7529414344d8?w=300',
      ],
      [
        'Творог с бананом',
        330,
        'Б 29 г · Ж 9 г · У 38 г',
        'https://images.unsplash.com/photo-1488477181946-6428a0291777?w=300',
      ],
      [
        'Тост с авокадо',
        410,
        'Б 14 г · Ж 19 г · У 46 г',
        'https://images.unsplash.com/photo-1541519227354-08fa5d50c44d?w=300',
      ],
    ],
    lunch: [
      [
        'Индейка с киноа',
        510,
        'Б 42 г · Ж 15 г · У 47 г',
        'https://images.unsplash.com/photo-1547592180-85f173990554?w=300',
      ],
      [
        'Паста с тунцом',
        560,
        'Б 36 г · Ж 16 г · У 62 г',
        'https://images.unsplash.com/photo-1473093295043-cdd812d0e601?w=300',
      ],
      [
        'Тофу с овощами',
        420,
        'Б 29 г · Ж 17 г · У 39 г',
        'https://images.unsplash.com/photo-1512621776951-a57141f2eefd?w=300',
      ],
    ],
    dinner: [
      [
        'Треска с овощами',
        390,
        'Б 38 г · Ж 12 г · У 27 г',
        'https://images.unsplash.com/photo-1519708227418-c8fd9a32b7a2?w=300',
      ],
      [
        'Куриный боул',
        470,
        'Б 40 г · Ж 13 г · У 45 г',
        'https://images.unsplash.com/photo-1512621776951-a57141f2eefd?w=300',
      ],
      [
        'Овощной суп',
        310,
        'Б 16 г · Ж 9 г · У 39 г',
        'https://images.unsplash.com/photo-1547592166-23ac45744acd?w=300',
      ],
    ],
    snack: [
      [
        'Протеиновый смузи',
        240,
        'Б 24 г · Ж 5 г · У 25 г',
        'https://images.unsplash.com/photo-1553530666-ba11a7da3888?w=300',
      ],
      [
        'Яблоко и йогурт',
        190,
        'Б 14 г · Ж 4 г · У 27 г',
        'https://images.unsplash.com/photo-1488477181946-6428a0291777?w=300',
      ],
    ],
  },
  metricInfo: {
    weight: [
      'Вес',
      '74.2 кг',
      'Целевой диапазон 72--76 кг. Динамика стабильная.',
      'В норме',
      'good',
    ],
    pulse: [
      'Пульс',
      '72 bpm',
      'Спокойный пульс соответствует текущему восстановлению.',
      'Норма',
      'good',
    ],
    sleep: [
      'Сон',
      '7.2 часа',
      'Для полной регенерации желательно 7.5--8 часов.',
      'Хорошо',
      'good',
    ],
    oxygen: [
      'Кислород',
      '98%',
      'Показатель находится в нормальном диапазоне.',
      'Норма',
      'good',
    ],
    hrv: [
      'ВСР',
      '64 мс',
      'Хорошая вариабельность сердечного ритма.',
      'Хорошо',
      'good',
    ],
    stress: [
      'Стресс',
      '32',
      'Уровень умеренный. Помогут дыхание и прогулка.',
      'Умеренный',
      'low',
    ],
    fat: [
      'Жир',
      '18.4%',
      'Показатель отмечен как ниже нормы в демо-данных.',
      'Ниже нормы',
      'low',
    ],
    muscle: [
      'Мышцы',
      '34.8 кг',
      'Показатель отмечен как выше нормы в демо-данных.',
      'Выше нормы',
      'high',
    ],
    protein: [
      'Белок',
      '16.2%',
      'Показатель находится в целевом диапазоне демо-данных.',
      'Норма',
      'good',
    ],
    bmi: [
      'ИМТ',
      '23.4',
      'Индекс массы тела в пределах нормы.',
      'Норма',
      'good',
    ],
  },
  achievements: [
    {
      name: '7 дней движения',
      done: true,
      icon: 'fa-fire',
      desc: 'Выполняй тренировки 7 дней подряд.',
    },
    {
      name: 'Белковый марафон',
      done: true,
      icon: 'fa-dumbbell',
      desc: 'Набирай 120+ г белка 5 дней.',
    },
    {
      name: 'Ранняя пташка',
      done: false,
      icon: 'fa-sun',
      desc: 'Просыпайся до 7:00 3 дня.',
    },
    {
      name: '10 000 шагов',
      done: false,
      icon: 'fa-shoe-prints',
      desc: 'Проходи 10 000 шагов в день.',
    },
    {
      name: 'Водный баланс',
      done: true,
      icon: 'fa-droplet',
      desc: 'Пей 2 литра воды 7 дней подряд.',
    },
    {
      name: 'Золотой ритм',
      done: false,
      icon: 'fa-trophy',
      desc: 'Достигни уровня 10.',
    },
    {
      name: 'Марафонец',
      done: false,
      icon: 'fa-medal',
      desc: '30 тренировок за месяц.',
    },
  ],
  activeDays: 12,
  survey: {
    step: 0,
    allergies: [],
    restrictions: [],
    goals: [],
    diet: '',
    activity: '',
    sleepHours: '',
    waterIntake: '',
    trainingDays: '',
    preferredWorkouts: [],
  },
  registered: false,
  guest: false,
  region: 'RU',
  surveyCompleted: false,
  surveyDeferred: false,
  chatConsent: false,
  chatSettings: {
    allowGuest: false,
    privateMessages: true,
    onlineStatus: false,
    readReceipts: true,
    notifications: true,
  },
  registrationData: { email: '', password: '', confirm: '', code: '' },
  chatContacts: [
    {
      name: 'Анна',
      last: 'Отлично! Завтра в 18:30?',
      time: '12:30',
      unread: 2,
      avatar: 'https://i.pravatar.cc/150?img=1',
      online: true,
      status: 'gym',
    },
    {
      name: 'Дмитрий',
      last: 'Сделал разминку, готов!',
      time: '10:15',
      unread: 0,
      avatar: 'https://i.pravatar.cc/150?img=2',
      online: true,
      status: 'pool',
    },
    {
      name: 'Елена',
      last: 'Спасибо за совет!',
      time: 'Вчера',
      unread: 1,
      avatar: 'https://i.pravatar.cc/150?img=3',
      online: false,
      status: 'offline',
    },
    {
      name: 'Сергей',
      last: 'Когда следующая тренировка?',
      time: 'Вчера',
      unread: 0,
      avatar: 'https://i.pravatar.cc/150?img=4',
      online: true,
      status: 'dnd',
    },
    {
      name: 'Ольга',
      last: 'Отличный результат!',
      time: '2 дня',
      unread: 0,
      avatar: 'https://i.pravatar.cc/150?img=5',
      online: false,
      status: 'offline',
    },
  ],
  chatMessages: [
    {
      from: 'Анна',
      text: 'Привет! Как прошла тренировка?',
      time: '12:20',
      sent: false,
    },
    {
      from: 'Я',
      text: 'Отлично! Сегодня силовая база.',
      time: '12:22',
      sent: true,
    },
    {
      from: 'Анна',
      text: 'Отлично! Завтра в 18:30?',
      time: '12:30',
      sent: false,
    },
  ],
  selectedChat: null,
  selectedCalendarDay: null,
  calendarEvents: {},
  customExercises: [],
  customMeals: [],
  trainingVideos: [
    {
      id: 1,
      title: 'Сильное тело за 20 минут',
      level: 'Средний',
      duration: '20 мин',
      equipment: ['Коврик', 'Гантели 2×5 кг'],
      description:
        'Комплекс без прыжков: приседания, отжимания, выпады, планка. Подходит для дома.',
      image:
        'https://images.unsplash.com/photo-1534438327276-14e5300c3a48?w=1000',
      type: 'home',
    },
    {
      id: 2,
      title: 'Мобильность после офиса',
      level: 'Лёгкий',
      duration: '15 мин',
      equipment: ['Коврик', 'Резинка'],
      description:
        'Упражнения для спины, таза и дыхания. Снимает напряжение после сидячей работы.',
      image: 'https://images.unsplash.com/photo-1544367567-0f2fcb009e0b?w=1000',
      type: 'home',
    },
    {
      id: 3,
      title: 'Техника плавания',
      level: 'Средний',
      duration: '35 мин',
      equipment: ['Очки', 'Шапочка', 'Доска для плавания'],
      description:
        'Разминка в воде, свободный стиль, спокойное плавание. Для бассейна.',
      image:
        'https://images.unsplash.com/photo-1530549387789-4c1017266635?w=1000',
      type: 'pool',
    },
    {
      id: 4,
      title: 'Кардио-интервалы',
      level: 'Высокий',
      duration: '25 мин',
      equipment: ['Коврик', 'Скакалка'],
      description: 'Интенсивные интервальные тренировки для сжигания жира.',
      image:
        'https://images.unsplash.com/photo-1518611012118-696072aa579a?w=1000',
      type: 'home',
    },
    {
      id: 5,
      title: 'Йога для начинающих',
      level: 'Лёгкий',
      duration: '30 мин',
      equipment: ['Коврик'],
      description: 'Мягкая йога для гибкости и расслабления.',
      image: 'https://images.unsplash.com/photo-1544367567-0f2fcb009e0b?w=1000',
      type: 'home',
    },
  ],
  progressGoals: {
    week: {
      label: 'Неделя',
      options: [
        { label: '3 дня', days: 3 },
        { label: '5 дней', days: 5 },
      ],
      selected: 0,
    },
    month: {
      label: 'Месяц',
      options: [
        { label: '10 дней', days: 10 },
        { label: '20 дней', days: 20 },
        { label: '30 дней', days: 30 },
      ],
      selected: 0,
    },
    year: {
      label: 'Год',
      options: [
        { label: '3 месяца', days: 90 },
        { label: '6 месяцев', days: 180 },
        { label: '9 месяцев', days: 270 },
        { label: '12 месяцев', days: 365 },
      ],
      selected: 0,
    },
  },
  currentProgressView: 'week',
  showDateModal: false,
  dateModalDate: null,
  selectedAchievement: null,
  selectedStory: null,
  storySeen: {},
  storyPaused: false,
  waterIntake: 0,
  waterDate: '',
  waterGoal: 8,
  completedWorkouts: [],
  dailyCompleted: 0,
  xp: 0,
  trainingMinutes: 0,
  caloriesBurned: 0,
  lastWaterUpdate: Date.now(),
  biometrics: [],
  bodyComposition: [],
  trainingPlans: [],
  selectedPlanId: null,
  mealsList: [],
  calendarEventsList: [],
  achievementsBackend: [],
  videos: [],
  conditions: [],
  menstrualCycles: [],
  profileLoaded: false,
  aiClassification: null,
  aiPlan: null,
  aiDiet: null,
  loading: {},
  lifeHacks: [
    {
      icon: 'fa-lightbulb',
      title: 'Пей воду до еды',
      text: 'Стакан воды за 20 минут до приёма пищи помогает контролировать аппетит и улучшает пищеварение.',
    },
    {
      icon: 'fa-clock',
      title: 'Правильный завтрак',
      text: 'Завтракай в течение часа после пробуждения --- это запускает метаболизм и даёт энергию на день.',
    },
    {
      icon: 'fa-bed',
      title: 'Сон и восстановление',
      text: 'Ложись спать до 23:00 --- гормон роста вырабатывается с 22:00 до 2:00, что важно для восстановления мышц.',
    },
    {
      icon: 'fa-utensils',
      title: 'Белок после тренировки',
      text: 'Употреби 20--30 г белка в течение 30 минут после тренировки для оптимального восстановления мышц.',
    },
    {
      icon: 'fa-heart',
      title: 'Сердечный ритм',
      text: 'Следи за пульсом во время тренировки. Оптимальная зона --- 60-80% от максимального.',
    },
    {
      icon: 'fa-moon',
      title: 'Режим сна',
      text: 'Стабильное время отхода ко сну улучшает качество восстановления.',
    },
  ],
  onlineUsers: 8,
  points: 1270,
  level: 5,
  streak: 12,
  maxStreak: 18,
  xpToNextLevel: 300,
  dailyQuests: [
    { name: 'Выпить 8 стаканов воды', done: false, xp: 20 },
    { name: 'Пройти 10 000 шагов', done: false, xp: 30 },
    { name: 'Выполнить тренировку', done: false, xp: 40 },
  ],
  instructions: {
    gym: '1. Разминка 5-10 мин (кардио + суставная гимнастика).\n2. Силовая тренировка: 3-4 подхода по 8-12 повторений на каждую группу мышц.\n3. Заминка: растяжка 5-10 мин.\n4. Пей воду во время тренировки (по глотку каждые 15 мин).\n5. Используй правильную технику, не гонись за весом.',
    pool: '1. Разминка на суше: 5 мин (махи, наклоны).\n2. В воде: 5 мин спокойного плавания для разогрева.\n3. Основная часть: интервалы (например, 4×50 м кроль с отдыхом 30 сек).\n4. Заминка: 5 мин спокойного плавания брассом.\n5. После бассейна: душ и растяжка.',
    hygiene:
      '1. Всегда принимай душ до и после тренировки.\n2. Используй индивидуальное полотенце и тапочки в бассейне.\n3. Дезинфицируй инвентарь после использования.\n4. Следи за чистотой спортивной одежды.\n5. Не занимайся при плохом самочувствии.',
  },
  friends: [
    {
      name: 'Анна',
      image: 'https://i.pravatar.cc/150?img=1',
      online: true,
      status: 'gym',
      bio: 'Бегу к своим целям маленькими шагами каждый день.',
      achievements: 8,
      workouts: 34,
      added: true,
    },
    {
      name: 'Дмитрий',
      image: 'https://i.pravatar.cc/150?img=2',
      online: true,
      status: 'pool',
      bio: 'Сила, дисциплина и хороший сон.',
      achievements: 12,
      workouts: 48,
      added: false,
    },
    {
      name: 'Елена',
      image: 'https://i.pravatar.cc/150?img=3',
      online: false,
      status: 'offline',
      bio: 'Йога и забота о себе в своём ритме.',
      achievements: 15,
      workouts: 61,
      added: false,
    },
    {
      name: 'Сергей',
      image: 'https://i.pravatar.cc/150?img=4',
      online: true,
      status: 'dnd',
      bio: 'Работаю над выносливостью и результатом.',
      achievements: 6,
      workouts: 27,
      added: false,
    },
    {
      name: 'Ольга',
      image: 'https://i.pravatar.cc/150?img=5',
      online: false,
      status: 'offline',
      bio: 'Здоровье --- это ежедневный выбор.',
      achievements: 10,
      workouts: 39,
      added: false,
    },
    {
      name: 'Алексей',
      image: 'https://i.pravatar.cc/150?img=6',
      online: true,
      status: 'gym',
      bio: 'Каждая тренировка --- это маленькая победа.',
      achievements: 5,
      workouts: 22,
      added: false,
    },
  ],
  stickers: [
    '💪',
    '🔥',
    '👏',
    '🏆',
    '❤️',
    '🥳',
    '💧',
    '🏋️',
    '🚀',
    '⭐',
    '🌈',
    '🎯',
  ],
  pinnedPlaces: [
    { name: 'Дом', icon: 'fa-house', color: '#6fae20', x: 20, y: 28 },
    { name: 'Зал', icon: 'fa-dumbbell', color: '#19d8da', x: 62, y: 62 },
    {
      name: 'Бассейн',
      icon: 'fa-person-swimming',
      color: '#19d8da',
      x: 78,
      y: 26,
    },
    { name: 'Работа', icon: 'fa-building', color: '#a78bfa', x: 42, y: 42 },
    {
      name: 'Учёба',
      icon: 'fa-graduation-cap',
      color: '#f5d45d',
      x: 82,
      y: 72,
    },
  ],
  hackIndex: 0,
  hackList: [
    [
      'fa-droplet',
      'Пей воду до еды',
      'Стакан воды за 20 минут до еды помогает поддерживать водный баланс.',
    ],
    [
      'fa-bed',
      'Сон --- часть прогресса',
      'Стабильный сон 7--9 часов помогает восстановлению мышц.',
    ],
    [
      'fa-person-running',
      'Не пропускай разминку',
      '5--10 минут разминки подготавливают суставы и мышцы к нагрузке.',
    ],
    [
      'fa-dumbbell',
      'Белок после тренировки',
      'Добавь источник белка в течение двух часов после занятия.',
    ],
    [
      'fa-heart',
      'Следи за пульсом',
      'Оптимальная зона пульса для жиросжигания --- 60-70% от максимального.',
    ],
    [
      'fa-moon',
      'Режим сна',
      'Стабильный график сна улучшает восстановление и продуктивность.',
    ],
  ],
  stories: [
    {
      title: 'Сила без перегруза',
      description: 'Тренировки под состояние',
      text: 'Слушай своё тело и выбирай нагрузку, которая приносит пользу.',
      image:
        'https://images.unsplash.com/photo-1517836357463-d25dfeac3438?w=900',
    },
    {
      title: 'Питание в ритме',
      description: 'КБЖУ без сложностей',
      text: 'Баланс питания помогает сохранять энергию и поддерживать результат.',
      image:
        'https://images.unsplash.com/photo-1490645935967-10de6ba17061?w=900',
    },
    {
      title: 'Улыбайся и двигайся',
      description: 'Энергия каждый день',
      text: 'Регулярное движение помогает укреплять тело и улучшать настроение.',
      image: 'https://images.unsplash.com/photo-1544367567-0f2fcb009e0b?w=900',
    },
    {
      title: 'Новая привычка',
      description: '12 дней серии',
      text: 'Каждый активный день приближает тебя к цели.',
      image:
        'https://images.unsplash.com/photo-1534438327276-14e5300c3a48?w=900',
    },
    {
      title: 'Сила духа',
      description: 'Ментальный фитнес',
      text: 'Крепкий дух помогает преодолевать любые физические барьеры.',
      image:
        'https://images.unsplash.com/photo-1518611012118-696072aa579a?w=900',
    },
    {
      title: 'Гармония тела',
      description: 'Баланс и гибкость',
      text: 'Сочетание силы и гибкости создаёт идеальный баланс для здоровья.',
      image: 'https://images.unsplash.com/photo-1544367567-0f2fcb009e0b?w=900',
    },
    {
      title: 'Водная стихия',
      description: 'Плавание и здоровье',
      text: 'Плавание укрепляет все группы мышц и улучшает дыхательную систему.',
      image:
        'https://images.unsplash.com/photo-1530549387789-4c1017266635?w=900',
    },
    {
      title: 'Утренний ритуал',
      description: 'Заряд энергии',
      text: 'Утренняя зарядка запускает обмен веществ и даёт энергию на весь день.',
      image:
        'https://images.unsplash.com/photo-1534438327276-14e5300c3a48?w=900',
    },
  ],
  nutritionData: {
    day: {
      labelRu: 'Сегодня',
      labelEn: 'Today',
      calories: 1680,
      goal: 2350,
      protein: 128,
      fats: 56,
      carbs: 185,
    },
    week: {
      labelRu: 'Неделя',
      labelEn: 'Week',
      calories: 11760,
      goal: 16450,
      protein: 896,
      fats: 392,
      carbs: 1295,
    },
    month: {
      labelRu: 'Месяц',
      labelEn: 'Month',
      calories: 50400,
      goal: 70500,
      protein: 3840,
      fats: 1680,
      carbs: 5550,
    },
  },
  trainingData: {
    day: {
      labelRu: 'День',
      labelEn: 'Day',
      completed: 2,
      goal: 3,
      calories: 420,
      minutes: 42,
      workouts: ['Силовая база', 'Разминка', 'Растяжка'],
    },
    week: {
      labelRu: 'Неделя',
      labelEn: 'Week',
      completed: 4,
      goal: 5,
      calories: 2140,
      minutes: 168,
      workouts: [
        'Силовая база',
        'Бассейн',
        'Мобильность',
        'Кардио',
        'Восстановление',
      ],
    },
    month: {
      labelRu: 'Месяц',
      labelEn: 'Month',
      completed: 16,
      goal: 20,
      calories: 12480,
      minutes: 720,
      workouts: [
        'Силовые тренировки',
        'Плавание',
        'Кардио',
        'Мобильность',
        'Восстановление',
      ],
    },
  },
  nutritionPeriod: 'day',
  trainingPeriod: 'day',
  videoIndex: 0,
  resetStep: 0,
  resetEmail: '',
  legalReturnScreen: null,
  __toast: null,
  __toastTimer: null,
};

function getInitialState() {
  try {
    const saved = JSON.parse(localStorage.getItem(KEY) || '{}');
    return {
      ...defaultState,
      screen: saved.screen || 'login',
      theme: saved.theme || 'dark',
      highContrast: saved.highContrast || false,
      language: saved.language || 'ru',
      calendarMonthIndex:
        saved.calendarMonthIndex !== undefined
          ? saved.calendarMonthIndex
          : new Date().getMonth(),
      calendarYear: saved.calendarYear || new Date().getFullYear(),
      profileImage: saved.profileImage || '',
      profile: {
        ...defaultState.profile,
        name: saved.profileName || defaultState.profile.name,
        gender: saved.gender || defaultState.profile.gender,
        email: saved.email || defaultState.profile.email,
        phone: saved.phone || defaultState.profile.phone,
        twoFactor: saved.twoFactor || defaultState.profile.twoFactor,
        backupCodesRemaining: saved.backupCodesRemaining || 0,
        devices: saved.devices || defaultState.profile.devices,
        bio: saved.bio || defaultState.profile.bio,
        status: saved.status || defaultState.profile.status,
        friends:
          saved.friendsCount !== undefined
            ? saved.friendsCount
            : defaultState.profile.friends,
      },
      calendarEvents: saved.calendarEvents || {},
      customExercises: saved.customExercises || [],
      customMeals: saved.customMeals || [],
      waterIntake: saved.waterIntake || 0,
      waterDate: saved.waterDate || '',
      points: saved.points || 1270,
      level: saved.level || 5,
      streak: saved.streak || 12,
      maxStreak: saved.maxStreak || 18,
      friends: saved.friends || defaultState.friends,
      pinnedPlaces: saved.pinnedPlaces || defaultState.pinnedPlaces,
      hackIndex: saved.hackIndex || 0,
      nutritionPeriod: saved.nutritionPeriod || 'day',
      trainingPeriod: saved.trainingPeriod || 'day',
      completedWorkouts: saved.completedWorkouts || [],
      dailyCompleted: saved.dailyCompleted || 0,
      xp: saved.xp || 0,
      trainingMinutes: saved.trainingMinutes || 0,
      caloriesBurned: saved.caloriesBurned || 0,
      biometrics: saved.biometrics || [],
      bodyComposition: saved.bodyComposition || [],
      trainingPlans: saved.trainingPlans || [],
      selectedPlanId: saved.selectedPlanId || null,
      mealsList: saved.mealsList || [],
      calendarEventsList: saved.calendarEventsList || [],
      achievementsBackend: saved.achievementsBackend || [],
      videos: saved.videos || [],
      conditions: saved.conditions || [],
      menstrualCycles: saved.menstrualCycles || [],
      profileLoaded: false,
      aiClassification: saved.aiClassification || null,
      aiPlan: saved.aiPlan || null,
      aiDiet: saved.aiDiet || null,
      loading: saved.loading || {},
      survey: saved.survey || defaultState.survey,
      surveyCompleted: saved.surveyCompleted || false,
      surveyDeferred: saved.surveyDeferred || false,
      registered: saved.registered || false,
      guest: saved.guest || false,
      region: saved.region || 'RU',
      chatConsent: saved.chatConsent || false,
      chatSettings: saved.chatSettings || defaultState.chatSettings,
      chatMessages: saved.chatMessages || defaultState.chatMessages,
      storySeen: saved.storySeen || {},
      twoFactorSetup: saved.twoFactorSetup || null,
      twoFactorTempToken: saved.twoFactorTempToken || null,
    };
  } catch {
    return { ...defaultState };
  }
}

const AppContext = createContext(null);

export function AppProvider({ children }) {
  const [state, setState] = useState(() => getInitialState());

  const save = useCallback(() => {
    try {
      localStorage.setItem(
        KEY,
        JSON.stringify({
          theme: state.theme,
          highContrast: state.highContrast,
          language: state.language,
          profileImage: state.profileImage,
          profileName: state.profile.name,
          gender: state.profile.gender,
          email: state.profile.email,
          phone: state.profile.phone,
          twoFactor: state.profile.twoFactor,
          assistantStyle: state.profile.assistantStyle,
          devices: state.profile.devices,
          bio: state.profile.bio,
          status: state.profile.status,
          friendsCount: state.profile.friends,
          calendarEvents: state.calendarEvents,
          customExercises: state.customExercises,
          customMeals: state.customMeals,
          waterIntake: state.waterIntake,
          waterDate: state.waterDate,
          points: state.points,
          level: state.level,
          streak: state.streak,
          maxStreak: state.maxStreak,
          friends: state.friends,
          pinnedPlaces: state.pinnedPlaces,
          hackIndex: state.hackIndex,
          nutritionPeriod: state.nutritionPeriod,
          trainingPeriod: state.trainingPeriod,
          calendarMonthIndex: state.calendarMonthIndex,
          calendarYear: state.calendarYear,
          completedWorkouts: state.completedWorkouts,
          dailyCompleted: state.dailyCompleted,
          xp: state.xp,
          trainingMinutes: state.trainingMinutes,
          caloriesBurned: state.caloriesBurned,
          biometrics: state.biometrics,
          bodyComposition: state.bodyComposition,
          trainingPlans: state.trainingPlans,
          selectedPlanId: state.selectedPlanId,
          mealsList: state.mealsList,
          calendarEventsList: state.calendarEventsList,
          achievementsBackend: state.achievementsBackend,
          videos: state.videos,
          conditions: state.conditions,
          menstrualCycles: state.menstrualCycles,
          aiClassification: state.aiClassification,
          aiPlan: state.aiPlan,
          aiDiet: state.aiDiet,
          survey: state.survey,
          surveyCompleted: state.surveyCompleted,
          surveyDeferred: state.surveyDeferred,
          registered: state.registered,
          guest: state.guest,
          region: state.region,
          chatConsent: state.chatConsent,
          chatSettings: state.chatSettings,
          chatMessages: state.chatMessages,
          storySeen: state.storySeen,
          twoFactorSetup: state.twoFactorSetup,
          twoFactorTempToken: state.twoFactorTempToken,
        })
      );
    } catch {
      // localStorage full or unavailable
    }
  }, [state]);

  useEffect(() => {
    save();
  }, [save]);

  useEffect(() => {
    document.documentElement.dataset.theme = state.theme;
    document.documentElement.dataset.highcontrast = state.highContrast
      ? 'on'
      : 'off';
    document.documentElement.lang = state.language === 'en' ? 'en' : 'ru';
  }, [state.theme, state.language, state.highContrast]);

  const update = useCallback((updates) => {
    setState((prev) => {
      if (typeof updates === 'function') {
        updates = updates(prev);
      }
      return { ...prev, ...updates };
    });
  }, []);

  const notify = useCallback(
    (message, duration = 2600) => {
      // Simple toast via state
      update({ __toast: message, __toastTimer: Date.now() });
      setTimeout(() => update({ __toast: null }), duration);
    },
    [update]
  );

  const doLogin = useCallback(async () => {
    const email = document.getElementById('loginEmail')?.value.trim();
    const pass = document.getElementById('loginPassword')?.value.trim();
    if (!email || !pass) {
      notify('Неверный email или пароль');
      return;
    }
    try {
      const data = await backendRequest('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password: pass }),
      });
      if (!data?.access_token || data?.status !== 'ok') {
        notify('Неверный email или пароль');
        return;
      }
      localStorage.setItem('fitpulse-access-token', data.access_token);
      if (data.refresh_token) {
        localStorage.setItem('fitpulse-refresh-token', data.refresh_token);
      }
      update({
        profile: { ...state.profile, email, password: pass },
        registered: true,
        guest: false,
        screen: 'home',
      });
      window.location.href = '/home';
    } catch {
      notify('Ошибка входа');
    }
  }, [state.profile, update, notify]);

  const logout = useCallback(async () => {
    try {
      await backendRequest('/api/v1/logout', { method: 'POST' });
    } catch {
      // ignore logout errors
    }
    localStorage.removeItem('fitpulse-access-token');
    localStorage.removeItem('fitpulse-refresh-token');
    update({ registered: false, guest: false, screen: 'login' });
    window.location.href = '/';
  }, [update]);

  const doRegister = useCallback(() => {
    const email = document.getElementById('regEmail')?.value.trim();
    const pass = document.getElementById('regPassword')?.value;
    const confirm = document.getElementById('regConfirm')?.value;
    if (!email || !pass || !confirm) {
      notify('Заполните все поля');
      return;
    }
    if (pass !== confirm) {
      notify('Пароли не совпадают');
      return;
    }
    if (pass.length < 6) {
      notify('Пароль должен содержать минимум 6 символов');
      return;
    }
    if (!document.getElementById('regConsent')?.checked) {
      notify('Подтвердите согласие с документами.');
      return;
    }
    update({
      guest: false,
      registrationData: { email, password: pass, confirm, code: '' },
      profile: { ...state.profile, email },
      registered: true,
      screen: 'login',
    });
    notify('Регистрация успешна. Проверьте email для подтверждения.');
  }, [state.profile, update, notify]);

  const forgotPassword = useCallback(async () => {
    update({ screen: 'reset', resetStep: 0 });
  }, [update]);

  const submitResetEmail = useCallback(async () => {
    const email = document.getElementById('resetEmail')?.value.trim();
    if (!email?.includes('@')) {
      notify('Введите корректный email');
      return;
    }
    try {
      await backendRequest('/api/v1/auth/forgot-password', {
        method: 'POST',
        body: JSON.stringify({ email }),
      });
      update({ resetEmail: email, resetStep: 1 });
      notify('Код отправлен на email');
    } catch {
      notify('Ошибка отправки кода');
    }
  }, [update, notify]);

  const submitResetCode = useCallback(() => {
    const code = document.getElementById('resetCode')?.value.trim();
    if (!code) {
      notify('Введите код');
      return;
    }
    update({ resetStep: 2 });
  }, [update, notify]);

  const submitNewPassword = useCallback(async () => {
    const pass = document.getElementById('newPassword')?.value || '';
    const confirmPass =
      document.getElementById('newPasswordConfirm')?.value || '';
    if (pass.length < 6) {
      notify('Пароль должен содержать минимум 6 символов');
      return;
    }
    if (pass !== confirmPass) {
      notify('Пароли не совпадают');
      return;
    }
    try {
      await backendRequest('/api/v1/auth/reset', {
        method: 'POST',
        body: JSON.stringify({
          email: state.resetEmail,
          code: document.getElementById('resetCode')?.value.trim(),
          new_password: pass,
        }),
      });
      update({
        profile: { ...state.profile, password: pass },
        screen: 'login',
      });
      notify('Пароль изменён. Теперь можно войти.');
    } catch {
      notify('Ошибка сброса пароля');
    }
  }, [state.profile, state.resetEmail, update, notify]);

  const continueAsGuest = useCallback(() => {
    update({ guest: true, registered: false, screen: 'home' });
    window.location.href = '/home';
  }, [update]);

  const socialLogin = useCallback(
    (provider) => {
      if (provider === 'google') {
        window.location.href = '/api/v1/auth/google';
        return;
      }
      notify(`Интеграция ${provider} не поддерживается`);
    },
    [notify]
  );

  const go = useCallback(
    (screen) => {
      if (
        state.screen === screen &&
        !state.selectedMeal &&
        !state.selectedMetric &&
        !state.showDateModal &&
        !state.selectedAchievement &&
        !state.selectedStory
      )
        return;
      update({
        screen,
        selectedMeal: null,
        selectedMetric: null,
        showDateModal: false,
        selectedAchievement: null,
        selectedStory: null,
        selectedChat: screen === 'chat' ? null : state.selectedChat,
      });
    },
    [
      state.screen,
      state.selectedMeal,
      state.selectedMetric,
      state.showDateModal,
      state.selectedAchievement,
      state.selectedStory,
      state.selectedChat,
      update,
    ]
  );

  const openLegal = useCallback(
    (screen) => {
      const defaultReturn = state.guest ? 'home' : 'login';
      update({
        legalReturnScreen:
          screen === 'privacy' || screen === 'terms'
            ? state.legalReturnScreen || defaultReturn
            : state.screen,
        screen,
      });
    },
    [state.screen, state.guest, state.legalReturnScreen, update]
  );

  const backFromLegal = useCallback(() => {
    const target = state.legalReturnScreen || (state.guest ? 'home' : 'login');
    update({ legalReturnScreen: null, screen: target });
  }, [state.legalReturnScreen, state.guest, update]);

  const toggleTheme = useCallback(() => {
    update({ theme: state.theme === 'dark' ? 'light' : 'dark' });
  }, [state.theme, update]);

  const toggleLanguage = useCallback(() => {
    update({ language: state.language === 'ru' ? 'en' : 'ru' });
  }, [state.language, update]);

  const toggleHighContrast = useCallback(() => {
    update({ highContrast: !state.highContrast });
  }, [state.highContrast, update]);

  const toggleTwoFactor = useCallback(() => {
    const next = !state.profile.twoFactor;
    if (
      next &&
      !confirm(
        'Включить двухфакторную аутентификацию? Для реальной защиты потребуется серверная проверка второго фактора.'
      )
    )
      return;
    update({ profile: { ...state.profile, twoFactor: next } });
  }, [state.profile, update]);

  const setup2FA = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/auth/2fa/setup', {
        method: 'POST',
      });
      if (data) {
        update({
          twoFactorSetup: {
            qrCodeUrl: data.qr_code_url || '',
            qrCodeBase64: data.qr_code_base64 || '',
            secret: data.secret || '',
            backupCodes: data.backup_codes || [],
          },
          screen: 'twofa-setup',
        });
      }
    } catch {
      notify('Не удалось настроить 2FA');
    }
  }, [update, notify]);

  const confirm2FA = useCallback(
    async (passcode, tempSecret, backupCodes) => {
      try {
        const data = await backendRequest('/api/v1/auth/2fa/confirm', {
          method: 'POST',
          body: JSON.stringify({
            passcode,
            temp_secret: tempSecret,
            backup_codes: backupCodes,
          }),
        });
        if (data?.success) {
          notify('2FA успешно настроена');
          update({ twoFactorSetup: null, screen: 'profile' });
        } else {
          notify(data?.message || 'Ошибка подтверждения 2FA');
        }
      } catch {
        notify('Не удалось подтвердить 2FA');
      }
    },
    [update, notify]
  );

  const verify2FA = useCallback(
    async (tempToken, passcode, isBackupCode = false) => {
      try {
        const data = await backendRequest('/api/v1/auth/2fa/verify', {
          method: 'POST',
          body: JSON.stringify({
            temp_token: tempToken,
            passcode,
            is_backup_code: isBackupCode,
          }),
        });
        if (data?.access_token) {
          localStorage.setItem('fitpulse-access-token', data.access_token);
          if (data?.refresh_token) {
            localStorage.setItem('fitpulse-refresh-token', data.refresh_token);
          }
          update({
            registered: true,
            guest: false,
            screen: 'home',
            twoFactorTempToken: null,
          });
        } else {
          notify('Неверный код 2FA');
        }
      } catch {
        notify('Ошибка проверки 2FA');
      }
    },
    [update, notify]
  );

  const disable2FA = useCallback(
    async (passcode) => {
      try {
        const data = await backendRequest('/api/v1/auth/2fa/disable', {
          method: 'POST',
          body: JSON.stringify({ passcode }),
        });
        if (data?.success) {
          update({ profile: { ...state.profile, twoFactor: false } });
          notify('2FA отключена');
        } else {
          notify(data?.message || 'Ошибка отключения 2FA');
        }
      } catch {
        notify('Не удалось отключить 2FA');
      }
    },
    [state.profile, update, notify]
  );

  const load2FAStatus = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/auth/2fa/status');
      if (data) {
        update({
          profile: {
            ...state.profile,
            twoFactor: data.enabled,
            backupCodesRemaining: data.backup_codes_remaining,
          },
        });
      }
    } catch {
      // ignore
    }
  }, [state.profile, update]);

  const showNotification = useCallback(() => {
    const msgs = [
      '💧 Не забудь выпить стакан воды!',
      '🍽️ Время обеда — 13:20',
      '🏋️ Тренировка в 18:30',
      '😴 Пора готовиться ко сну',
    ];
    notify(msgs.join('\n'));
  }, [notify]);

  const drinkWater = useCallback(() => {
    if (state.waterIntake < state.waterGoal) {
      const newIntake = state.waterIntake + 1;
      const newState = {
        ...state,
        waterIntake: newIntake,
        lastWaterUpdate: Date.now(),
      };
      if (newIntake === state.waterGoal) {
        notify('🎉 Отлично! Ты выполнил норму воды на сегодня!');
        const ach = newState.achievements.find(
          (a) => a.name === 'Водный баланс'
        );
        if (ach && !ach.done) {
          ach.done = true;
          notify('⭐ Новое достижение: Водный баланс!');
        }
        newState.points += 30;
        if (newState.points >= newState.level * 1000) {
          newState.level++;
          newState.points = 0;
          notify(`🎉 Новый уровень! Уровень ${newState.level}`);
        }
      }
      update(newState);
    } else {
      notify('Ты уже выполнил норму воды на сегодня! 💧');
    }
  }, [state, update, notify]);

  const changeCalendar = useCallback(
    (direction) => {
      let m = state.calendarMonthIndex + Number(direction || 0);
      let y = state.calendarYear;
      while (m < 0) {
        m += 12;
        y--;
      }
      while (m > 11) {
        m -= 12;
        y++;
      }
      if (!isDateAllowed(y, m)) {
        notify('Нельзя выбрать дату раньше августа 2025 года');
        return;
      }
      update({
        calendarMonthIndex: m,
        calendarYear: y,
        selectedCalendarDay: null,
      });
    },
    [state.calendarMonthIndex, state.calendarYear, update, notify]
  );

  const quickAddEvent = useCallback(
    (type) => {
      const d = new Date();
      const colors = {
        gym: '#6fae20',
        pool: '#19d8da',
        recovery: '#ff6375',
        work: '#a78bfa',
        study: '#f5d45d',
        note: '#a78bfa',
      };
      const labels = {
        gym: 'Тренировка в зале',
        pool: 'Бассейн',
        recovery: 'Восстановление',
        work: 'Работа',
        study: 'Учёба',
        note: 'Заметка',
      };
      const key = getDateKey(d.getFullYear(), d.getMonth(), d.getDate());
      const newEvents = { ...state.calendarEvents };
      if (!newEvents[key]) newEvents[key] = [];
      newEvents[key] = [
        ...newEvents[key],
        {
          title: labels[type] || type,
          type,
          date: `${d.getDate()} ${getMonthName(d.getMonth(), state.language)}`,
          time: '19:00',
          color: colors[type] || '#19d8da',
        },
      ];
      update({ calendarEvents: newEvents });
    },
    [state.calendarEvents, state.language, update]
  );

  const setNutritionPeriod = useCallback(
    (period) => {
      update({ nutritionPeriod: period });
    },
    [update]
  );

  const setTrainingPeriod = useCallback(
    (period) => {
      update({ trainingPeriod: period });
    },
    [update]
  );

  const openMeal = useCallback(
    (slot) => {
      update({ selectedMeal: slot });
    },
    [update]
  );

  const selectMeal = useCallback(
    (slot, title, kcal, macros, image) => {
      const newMeals = { ...state.meals };
      newMeals[slot] = { ...newMeals[slot], title, kcal, macros, image };
      const d = new Date();
      const key = getDateKey(d.getFullYear(), d.getMonth(), d.getDate());
      const newEvents = { ...state.calendarEvents };
      if (!newEvents[key]) newEvents[key] = [];
      newEvents[key] = [
        ...newEvents[key],
        {
          type: 'food',
          title,
          info: `${kcal} ${t('ккал', 'kcal')} · ${macros}`,
          time: newMeals[slot].time,
          date: `${d.getDate()} ${getMonthName(d.getMonth(), state.language)}`,
          color: '#f5d45d',
        },
      ];
      update({
        meals: newMeals,
        selectedMeal: null,
        calendarEvents: newEvents,
      });
    },
    [state.meals, state.calendarEvents, state.language, update]
  );

  const saveCustomMeal = useCallback(() => {
    const name =
      document.getElementById('customName')?.value.trim() || 'Своё блюдо';
    const kcal = Number(document.getElementById('customKcal')?.value) || 0;
    const macros =
      document.getElementById('customMacros')?.value || 'Б 0 г · Ж 0 г · У 0 г';
    const slot = document.getElementById('customSlot')?.value || 'breakfast';
    const newMeals = { ...state.meals };
    newMeals[slot] = { ...newMeals[slot], title: name, kcal, macros };
    const d = new Date();
    const key = getDateKey(d.getFullYear(), d.getMonth(), d.getDate());
    const newEvents = { ...state.calendarEvents };
    if (!newEvents[key]) newEvents[key] = [];
    newEvents[key] = [
      ...newEvents[key],
      {
        type: 'food',
        title: name,
        info: `${kcal} ${t('ккал', 'kcal')} · ${macros}`,
        time: newMeals[slot].time,
        date: `${d.getDate()} ${getMonthName(d.getMonth(), state.language)}`,
        color: '#f5d45d',
      },
    ];
    update({ meals: newMeals, calendarEvents: newEvents });
    notify('Блюдо добавлено в меню и календарь');
  }, [state.meals, state.calendarEvents, state.language, update, notify]);

  const askAI = useCallback(
    (_text) => {
      update({ screen: 'ai' });
      // AI answer will be handled in the AI screen
    },
    [update]
  );

  const openMetric = useCallback(
    (metric) => {
      const info = state.metricInfo?.[metric];
      if (!info) return;
      update({
        selectedMetric: {
          key: metric,
          name: info[0],
          value: info[1],
          desc: info[2],
          status: info[3],
          tone: info[4],
        },
      });
    },
    [state.metricInfo, update]
  );

  const addWeight = useCallback(() => {
    const newHistory = [
      ...state.weightHistory,
      [t('Сегодня', 'Today'), '74.0 кг'],
    ];
    update({ weightHistory: newHistory });
  }, [state.weightHistory, update]);

  const toggleWorkout = useCallback(
    (id) => {
      update({ selectedWorkout: state.selectedWorkout === id ? null : id });
    },
    [state.selectedWorkout, update]
  );

  const startWorkout = useCallback(
    async (title) => {
      const completed = state.completedWorkouts || [];
      if (completed.includes(title)) {
        notify('Эта тренировка уже отмечена как выполненная');
        return;
      }
      try {
        await backendRequest('/api/v1/training/complete', {
          method: 'POST',
          body: JSON.stringify({
            plan_id: 'default',
            workout_id: title,
            rating: 5,
            feedback: 'Completed',
          }),
        });
        const newCompleted = [...completed, title];
        const newDaily = (state.dailyCompleted || 0) + 1;
        const td = state.trainingData?.day;
        const newTrainingData = td
          ? {
              ...td,
              completed: Math.min(td.goal || 3, (td.completed || 0) + 1),
              minutes: (td.minutes || 0) + 42,
              calories: (td.calories || 0) + 420,
            }
          : td;
        update({
          completedWorkouts: newCompleted,
          dailyCompleted: newDaily,
          points: (state.points || 0) + 40,
          xp: (state.xp || 0) + 40,
          trainingMinutes: (state.trainingMinutes || 0) + 42,
          caloriesBurned: (state.caloriesBurned || 0) + 420,
          trainingData: newTrainingData
            ? { ...state.trainingData, day: newTrainingData }
            : state.trainingData,
        });
        notify(`${title}\n✓ Тренировка завершена. +40 XP`);
      } catch {
        notify('Ошибка завершения тренировки');
      }
    },
    [
      state.completedWorkouts,
      state.dailyCompleted,
      state.points,
      state.xp,
      state.trainingMinutes,
      state.caloriesBurned,
      state.trainingData,
      update,
      notify,
    ]
  );

  const editPlace = useCallback(
    (index) => {
      const p = state.pinnedPlaces[index];
      if (!p) return;
      const name = prompt(t('Название точки', 'Place name'), p.name);
      if (name === null) return;
      const newPlaces = [...state.pinnedPlaces];
      newPlaces[index] = {
        ...newPlaces[index],
        name: name.trim() || newPlaces[index].name,
        x: Math.max(
          3,
          Math.min(
            97,
            Number(prompt(t('X 0–100', 'X 0–100'), newPlaces[index].x)) ||
              newPlaces[index].x
          )
        ),
        y: Math.max(
          3,
          Math.min(
            90,
            Number(prompt(t('Y 0–100', 'Y 0–100'), newPlaces[index].y)) ||
              newPlaces[index].y
          )
        ),
      };
      update({ pinnedPlaces: newPlaces });
    },
    [state.pinnedPlaces, update]
  );

  const selectPlace = useCallback(
    (place) => {
      update({ selectedPlace: place });
    },
    [update]
  );

  const addPlace = useCallback(() => {
    const name = prompt(
      t('Название точки', 'Place name'),
      t('Новое место', 'New place')
    );
    if (!name?.trim()) return;
    const icon = (
      prompt(
        t('Иконка Font Awesome', 'Font Awesome icon'),
        'fa-location-dot'
      ) || 'fa-location-dot'
    ).trim();
    const color =
      prompt(
        t('Цвет точки (например #19d8da)', 'Pin color (e.g. #19d8da)'),
        '#19d8da'
      ) || '#19d8da';
    const x = Math.max(
      3,
      Math.min(
        97,
        Number(
          prompt(
            t('Положение по горизонтали 0–100', 'Horizontal position 0–100'),
            '50'
          )
        ) || 50
      )
    );
    const y = Math.max(
      3,
      Math.min(
        90,
        Number(
          prompt(
            t('Положение по вертикали 0–100', 'Vertical position 0–100'),
            '50'
          )
        ) || 50
      )
    );
    const newPlaces = [
      ...state.pinnedPlaces,
      { name: name.trim(), icon, color, x, y },
    ];
    update({ pinnedPlaces: newPlaces });
  }, [state.pinnedPlaces, update]);

  const openCalendarPage = useCallback(
    (resetToToday = true) => {
      if (resetToToday) {
        const now = new Date();
        update({
          calendarMonthIndex: now.getMonth(),
          calendarYear: now.getFullYear(),
        });
      }
    },
    [update]
  );

  const selectCalendarDay = useCallback(
    (day) => {
      update({
        selectedCalendarDay: day,
        dateModalDate: day,
        showDateModal: true,
      });
    },
    [update]
  );

  const closeDateModal = useCallback(() => {
    update({ showDateModal: false, dateModalDate: null });
  }, [update]);

  const addEventToDay = useCallback(
    (day) => {
      update({
        selectedCalendarDay: day,
        dateModalDate: day,
        showDateModal: true,
      });
    },
    [update]
  );

  const addEvent = useCallback(
    async (eventData = {}) => {
      const day = state.selectedCalendarDay || new Date().getDate();
      const type =
        eventData.type ||
        document.getElementById('eventType')?.value ||
        'training';
      const title =
        eventData.title ||
        document.getElementById('eventTitle')?.value.trim() ||
        t('Новое событие', 'New event');
      const time =
        eventData.time ||
        document.getElementById('eventTime')?.value ||
        '19:00';
      const description =
        eventData.description ||
        document.getElementById('eventDescription')?.value.trim() ||
        '';
      const dateStr = `${state.calendarYear}-${String(state.calendarMonthIndex + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
      try {
        const data = await backendRequest('/api/v1/calendar/events', {
          method: 'POST',
          body: JSON.stringify({ title, date: dateStr, type, description }),
        });
        if (data?.event) {
          const newEvents = { ...state.calendarEvents };
          const dateLabel = `${day} ${getMonthName(state.calendarMonthIndex, state.language)}`;
          if (!newEvents[dateLabel]) newEvents[dateLabel] = [];
          newEvents[dateLabel] = [
            ...newEvents[dateLabel],
            {
              id: data.event.id,
              title: data.event.title,
              type: data.event.type,
              info: description || type,
              time,
              date: dateLabel,
              color:
                {
                  training: '#6fae20',
                  food: '#f5d45d',
                  recovery: '#ff6375',
                  meeting: '#a78bfa',
                  note: '#19d8da',
                }[type] || '#19d8da',
            },
          ];
          const newList = [
            ...(state.calendarEventsList || []),
            { ...data.event, dateLabel },
          ];
          update({
            calendarEvents: newEvents,
            calendarEventsList: newList,
            showDateModal: false,
            dateModalDate: null,
          });
        }
      } catch {
        notify('Ошибка добавления события');
      }
    },
    [
      state.selectedCalendarDay,
      state.calendarMonthIndex,
      state.calendarYear,
      state.calendarEvents,
      state.language,
      update,
      notify,
      t,
    ]
  );

  const updateEvent = useCallback(
    async (eventId, title, date, type, description) => {
      try {
        const data = await backendRequest(
          `/api/v1/calendar/events/${eventId}`,
          {
            method: 'PUT',
            body: JSON.stringify({ title, date, type, description }),
          }
        );
        if (data?.event) {
          const newList = (state.calendarEventsList || []).map((e) =>
            e.id === eventId ? { ...e, ...data.event } : e
          );
          update({ calendarEventsList: newList });
          notify('Событие обновлено');
        }
      } catch {
        notify('Ошибка обновления события');
      }
    },
    [state.calendarEventsList, update, notify]
  );

  const deleteCalendarEvent = useCallback(
    async (keyOrEventId, index) => {
      let eventId = keyOrEventId;

      if (typeof index === 'number' && typeof keyOrEventId === 'string') {
        const list = state.calendarEvents[keyOrEventId];
        if (!list || index < 0 || index >= list.length) return;
        eventId = list[index].id;
      }

      try {
        await backendRequest(`/api/v1/calendar/events/${eventId}`, {
          method: 'DELETE',
        });
        const newList = (state.calendarEventsList || []).filter(
          (e) => e.id !== eventId
        );
        const newEvents = { ...state.calendarEvents };
        for (const key of Object.keys(newEvents)) {
          newEvents[key] = newEvents[key].filter((e) => e.id !== eventId);
          if (!newEvents[key].length) delete newEvents[key];
        }
        update({
          calendarEvents: newEvents,
          calendarEventsList: newList,
          showDateModal: false,
          dateModalDate: null,
        });
      } catch {
        notify('Ошибка удаления события');
      }
    },
    [state.calendarEvents, state.calendarEventsList, update, notify]
  );

  const sendAI = useCallback(async () => {
    const input = document.getElementById('aiInput');
    const text = input?.value.trim();
    if (!text) return;
    const userMsg = { id: `${Date.now()}-u`, type: 'user', text };
    const newMessages = [...state.messages, userMsg];
    update({ messages: newMessages });
    if (input) input.value = '';
    try {
      const data = await backendRequest('/api/v1/ml/chat', {
        method: 'POST',
        body: JSON.stringify({ message: text }),
      });
      if (data) {
        const aiMsg = {
          id: `${Date.now()}-a`,
          type: 'ai',
          text: data.message || 'Ответ получен',
          classification: data.classification || null,
          plan: data.plan || null,
          diet: data.diet || null,
          timestamp: data.timestamp,
        };
        update({
          messages: [...newMessages, aiMsg],
          aiClassification: data.classification || null,
          aiPlan: data.plan || null,
          aiDiet: data.diet || null,
        });
      }
    } catch {
      update({
        messages: [
          ...newMessages,
          {
            id: `${Date.now()}-e`,
            type: 'ai',
            text: 'Ошибка подключения к AI сервису',
          },
        ],
      });
    }
  }, [state.messages, update]);

  const quickAI = useCallback(
    async (text) => {
      const newMessages = [
        ...state.messages,
        { id: `${Date.now()}-u`, type: 'user', text },
      ];
      update({ messages: newMessages });
      try {
        const data = await backendRequest('/api/v1/ml/chat', {
          method: 'POST',
          body: JSON.stringify({ message: text }),
        });
        if (data) {
          const aiMsg = {
            id: `${Date.now()}-a`,
            type: 'ai',
            text: data.message || 'Ответ получен',
            classification: data.classification || null,
            plan: data.plan || null,
            diet: data.diet || null,
            timestamp: data.timestamp,
          };
          update({
            messages: [...newMessages, aiMsg],
            aiClassification: data.classification || null,
            aiPlan: data.plan || null,
            aiDiet: data.diet || null,
          });
        }
      } catch {
        update({
          messages: [
            ...newMessages,
            {
              id: `${Date.now()}-e`,
              type: 'ai',
              text: 'Ошибка подключения к AI сервису',
            },
          ],
        });
      }
    },
    [state.messages, update]
  );

  const editProfile = useCallback(async () => {
    const p = state.profile;
    const name = prompt(t('Имя', 'Name'), p.name);
    if (name === null) return;
    const status = prompt(
      t('Статус', 'Status'),
      p.status || t('Активен', 'Active')
    );
    if (status === null) return;
    const bio = prompt(t('О себе', 'About'), p.bio || '');
    if (bio === null) return;
    const phone = prompt(t('Телефон', 'Phone'), p.phone || '');
    if (phone === null) return;
    try {
      await backendRequest('/api/v1/profile', {
        method: 'PUT',
        body: JSON.stringify({
          full_name: name,
          gender: p.gender,
          phone,
          bio,
          status,
        }),
      });
      update({
        profile: {
          ...p,
          name: name.trim() || p.name,
          status: status.trim(),
          bio: bio.trim(),
          phone: phone.trim(),
        },
      });
      notify('Профиль обновлён');
    } catch {
      notify('Ошибка обновления профиля');
    }
  }, [state.profile, update, notify, t]);

  const changePassword = useCallback(() => {
    const oldPass = document.getElementById('oldPass')?.value || '';
    const np = document.getElementById('newPass')?.value || '';
    const np2 = document.getElementById('newPass2')?.value || '';
    if (!oldPass) {
      notify('Введите старый пароль');
      return;
    }
    if (np.length < 6) {
      notify('Пароль должен содержать минимум 6 символов');
      return;
    }
    if (np !== np2) {
      notify('Пароли не совпадают');
      return;
    }
    update({ profile: { ...state.profile, password: np } });
    notify('Пароль изменён');
  }, [state.profile, update, notify]);

  const deleteAccount = useCallback(() => {
    if (
      !confirm(
        t(
          'Вы уверены, что хотите удалить аккаунт? Это действие необратимо.',
          'Are you sure you want to delete your account? This action is irreversible.'
        )
      )
    )
      return;
    localStorage.removeItem(KEY);
    notify(
      t(
        'Аккаунт удалён. Локальные данные очищены.',
        'Account deleted. Local data has been cleared.'
      )
    );
    update({ registered: false, guest: false, screen: 'login' });
  }, [update, notify]);

  const searchFriends = useCallback(() => {
    const q = prompt(t('Имя или ID пользователя', 'User name or ID'), '');
    if (!q) return;
    const matches = (state.friends || []).filter(
      (f) =>
        String(f.name || '')
          .toLowerCase()
          .includes(q.toLowerCase()) ||
        String(f.id || '')
          .toLowerCase()
          .includes(q)
    );
    if (!matches.length) {
      notify(t('Ничего не найдено', 'Nothing found'));
      return;
    }
    const f = matches[0];
    f.added = !f.added;
    update({
      friends: [...(state.friends || [])],
      profile: {
        ...state.profile,
        friends: (state.friends || []).filter((x) => x.added).length,
      },
    });
  }, [state.friends, state.profile, update, notify]);

  const showDevices = useCallback(() => {
    const name = prompt(
      t(
        'Новое устройство (оставьте пустым для удаления)',
        'New device (leave empty to remove)'
      ),
      ''
    );
    if (name === null) return;
    const devices = [...(state.profile.devices || [])];
    if (name.trim()) devices.push(name.trim());
    else if (devices.length) devices.pop();
    update({ profile: { ...state.profile, devices } });
  }, [state.profile, update]);

  const showRestrictions = useCallback(() => {
    const a = (state.survey?.allergies || []).join(', ') || t('нет', 'none');
    const r = (state.survey?.restrictions || []).join(', ') || t('нет', 'none');
    notify(
      `${t('Аллергии', 'Allergies')}: ${a}\n${t('Ограничения', 'Restrictions')}: ${r}`
    );
  }, [state.survey, notify]);

  const chatSettings = useCallback(() => {
    // For now, just a simple notify. In a real app, this would open a modal.
    notify(
      t(
        'Настройки чата: приватность, статус, уведомления',
        'Chat settings: privacy, status, notifications'
      )
    );
  }, [notify]);

  const nutritionSettings = useCallback(() => {
    notify(
      `${t('Настройки питания', 'Nutrition settings')}: ${t('цель', 'goal')} --- улучшение формы, 2 350 ${t('ккал', 'kcal')}, ${t('белок', 'protein')} 120--145 г, ${t('аллергии', 'allergies')} ${(state.survey?.allergies || []).join(', ') || t('не указаны', 'not specified')}`
    );
  }, [state.survey, notify]);

  const resumeSurvey = useCallback(() => {
    update({ screen: 'survey', surveyDeferred: false });
  }, [update]);

  const scrollStories = useCallback((_dir) => {
    // Handled by StoryCarousel component
  }, []);

  const showAllStories = useCallback(() => {
    if (!state.stories?.length) return;
    update({ selectedStory: state.stories[0] });
  }, [state.stories, update]);

  const openStory = useCallback(
    (index) => {
      if (!state.stories[index]) return;
      update({
        selectedStory: state.stories[index],
        storySeen: { ...state.storySeen, [index]: true },
      });
    },
    [state.stories, state.storySeen, update]
  );

  const closeStory = useCallback(() => {
    update({ selectedStory: null, storyPaused: false });
  }, [update]);

  const nextStory = useCallback(() => {
    const i = state.stories.indexOf(state.selectedStory);
    if (i < state.stories.length - 1) {
      update({
        selectedStory: state.stories[i + 1],
        storySeen: { ...state.storySeen, [i + 1]: true },
      });
    } else {
      closeStory();
    }
  }, [state.stories, state.selectedStory, state.storySeen, update, closeStory]);

  const prevStory = useCallback(() => {
    const i = state.stories.indexOf(state.selectedStory);
    if (i > 0) {
      update({
        selectedStory: state.stories[i - 1],
        storySeen: { ...state.storySeen, [i - 1]: true },
      });
    }
  }, [state.stories, state.selectedStory, state.storySeen, update]);

  const addCustomExercise = useCallback(() => {
    const name = document.getElementById('customExerciseName')?.value.trim();
    const sets = document.getElementById('customExerciseSets')?.value.trim();
    if (!name || !sets) {
      notify(t('Заполните все поля', 'Please fill all fields'));
      return;
    }
    update({
      customExercises: [...(state.customExercises || []), { name, sets }],
    });
    if (document.getElementById('customExerciseName'))
      document.getElementById('customExerciseName').value = '';
    if (document.getElementById('customExerciseSets'))
      document.getElementById('customExerciseSets').value = '';
  }, [state.customExercises, update, notify]);

  const removeCustomExercise = useCallback(
    (index) => {
      if (!confirm(t('Удалить упражнение?', 'Delete exercise?'))) return;
      update({
        customExercises: (state.customExercises || []).filter(
          (_, i) => i !== index
        ),
      });
    },
    [state.customExercises, update]
  );

  const openVideoModal = useCallback(
    (id) => {
      const video = state.trainingVideos.find((v) => v.id === id);
      if (!video) return;
      update({
        videoIndex: state.trainingVideos.indexOf(video),
        selectedVideoId: id,
      });
    },
    [state.trainingVideos, update]
  );

  const closeVideoModal = useCallback(() => {
    update({ selectedVideoId: null });
  }, [update]);

  const changeVideoModal = useCallback(
    (direction) => {
      const total = state.trainingVideos.length;
      const newIndex = (state.videoIndex + direction + total) % total;
      update({
        videoIndex: newIndex,
        selectedVideoId: state.trainingVideos[newIndex]?.id,
      });
    },
    [state.trainingVideos, state.videoIndex, update]
  );

  const connectVideo = useCallback(() => {
    const video = state.trainingVideos.find(
      (v) => v.id === state.selectedVideoId
    );
    if (video) {
      notify(
        `${video.title}\n\n${t('Это демонстрационный видео-урок. Реальный видеосервис подключается через backend/API.', 'This is a demo video lesson. A real video service is connected through the backend/API.')}`
      );
    }
  }, [state.trainingVideos, state.selectedVideoId, notify]);

  const toggleFriendAdded = useCallback(
    (value) => {
      const index =
        typeof value === 'number'
          ? value
          : (state.friends || []).findIndex(
              (f) => f.name === value || String(f.id || '') === String(value)
            );
      const f = (state.friends || [])[index];
      if (!f) return;
      const newFriends = [...(state.friends || [])];
      newFriends[index] = { ...f, added: !f.added };
      update({
        friends: newFriends,
        profile: {
          ...state.profile,
          friends: newFriends.filter((x) => x.added).length,
        },
      });
    },
    [state.friends, state.profile, update]
  );

  const openFriendProfile = useCallback(
    (friendOrName) => {
      let friend;
      if (typeof friendOrName === 'string') {
        friend = (state.friends || []).find((f) => f.name === friendOrName);
      } else {
        friend = friendOrName;
      }
      if (!friend) {
        notify(t('Пользователь не найден', 'User not found'));
        return;
      }
      update({ selectedFriend: friend });
    },
    [state.friends, update, notify]
  );

  const showAchievement = useCallback(
    (name) => {
      const ach = (state.achievements || []).find((a) => a.name === name);
      if (ach) update({ selectedAchievement: ach });
    },
    [state.achievements, update]
  );

  const showAchievementsList = useCallback(() => {
    update({ selectedAchievement: null });
  }, [update]);

  const surveySelect = useCallback(
    (key, value, isMulti) => {
      const survey = { ...state.survey };
      if (isMulti) {
        const arr = [...(survey[key] || [])];
        const idx = arr.indexOf(value);
        if (idx > -1) arr.splice(idx, 1);
        else arr.push(value);
        survey[key] = arr;
      } else {
        survey[key] = value;
      }
      update({ survey, surveyDeferred: false });
    },
    [state.survey, update]
  );

  const surveyNext = useCallback(() => {
    const steps = [
      'allergies',
      'restrictions',
      'goals',
      'diet',
      'activity',
      'sleepHours',
      'waterIntake',
      'preferredWorkouts',
      'trainingDays',
    ];
    const key = steps[state.survey.step];
    const val = state.survey[key];
    if (Array.isArray(val) && val.length === 0) {
      notify(t('Выберите хотя бы один вариант', 'Select at least one option'));
      return;
    }
    if (typeof val === 'string' && !val) {
      notify(t('Выберите один вариант', 'Select one option'));
      return;
    }
    if (state.survey.step < steps.length - 1) {
      update({ survey: { ...state.survey, step: state.survey.step + 1 } });
    } else {
      update({ surveyCompleted: true, surveyDeferred: false, screen: 'home' });
    }
  }, [state.survey, update, notify]);

  const surveyPrev = useCallback(() => {
    if (state.survey.step > 0) {
      update({ survey: { ...state.survey, step: state.survey.step - 1 } });
    }
  }, [state.survey, update]);

  const deferSurvey = useCallback(() => {
    update({ surveyDeferred: true, screen: 'home' });
  }, [update]);

  const openChat = useCallback(
    (name) => {
      update({ selectedChat: name, screen: 'chatDetail' });
    },
    [update]
  );

  const sendChat = useCallback(() => {
    if (state.guest) {
      notify(
        t(
          'В гостевом режиме личный чат недоступен.',
          'Private chat is unavailable in guest mode.'
        )
      );
      return;
    }
    const input = document.getElementById('chatInput');
    const text = input?.value.trim();
    if (!text) return;
    const time = new Date().toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
    });
    update({
      chatMessages: [
        ...(state.chatMessages || []),
        { from: 'Я', text, time, sent: true },
      ],
      chatConsent: true,
    });
    if (input) input.value = '';
    setTimeout(() => {
      update({
        chatMessages: [
          ...(state.chatMessages || []),
          {
            from: state.selectedChat || 'Анна',
            text: t(
              'Отлично! Продолжай в том же духе 💪',
              'Great! Keep it up 💪'
            ),
            time: new Date().toLocaleTimeString([], {
              hour: '2-digit',
              minute: '2-digit',
            }),
            sent: false,
          },
        ],
      });
    }, 800);
  }, [state.guest, state.chatMessages, state.selectedChat, update, notify]);

  const saveChatSettings = useCallback(() => {
    const onlineStatus = !!document.getElementById('csOnline')?.checked;
    const readReceipts = !!document.getElementById('csRead')?.checked;
    const notifications = !!document.getElementById('csNotify')?.checked;
    const chatConsent = !!document.getElementById('csConsent')?.checked;
    update({
      chatSettings: {
        ...state.chatSettings,
        onlineStatus,
        readReceipts,
        notifications,
      },
      chatConsent,
    });
  }, [state.chatSettings, update]);

  const addStickerToChat = useCallback((sticker) => {
    const input = document.getElementById('chatInput');
    if (input) {
      input.value += sticker;
      input.focus();
    }
  }, []);

  const uploadPhoto = useCallback(
    (event) => {
      const file = event.target.files?.[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = () => {
        update({ profileImage: reader.result });
      };
      reader.readAsDataURL(file);
    },
    [update]
  );

  const setGoal = useCallback(
    (type, index) => {
      if (state.progressGoals[type]) {
        const opt = state.progressGoals[type].options[index];
        notify(
          `${t('Цель', 'Goal')}: ${opt.label} (${opt.days} ${t('дней', 'days')})`
        );
        const newGoals = { ...state.progressGoals };
        newGoals[type] = { ...newGoals[type], selected: index };
        update({ progressGoals: newGoals });
      }
    },
    [state.progressGoals, update, notify]
  );

  const askAIForDay = useCallback(
    (day) => {
      update({ screen: 'ai', selectedCalendarDay: day });
    },
    [update]
  );

  const closeMetric = useCallback(() => {
    update({ selectedMetric: null });
  }, [update]);

  const closeMeal = useCallback(() => {
    update({ selectedMeal: null });
  }, [update]);

  const closeAchievement = useCallback(() => {
    update({ selectedAchievement: null });
  }, [update]);

  const loadProfile = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/profile');
      if (data?.profile) {
        const p = data.profile;
        update({
          profile: {
            ...state.profile,
            name: p.full_name || p.name || state.profile.name,
            email: p.email || state.profile.email,
            gender: p.gender || state.profile.gender,
            phone: p.phone || state.profile.phone,
            bio: p.bio || state.profile.bio,
            status: p.status || state.profile.status,
          },
          profileLoaded: true,
        });
      } else {
        update({ profileLoaded: true });
      }
    } catch {
      update({ profileLoaded: true });
    }
  }, [state.profile, update]);

  const saveProfile = useCallback(
    async (profileData) => {
      try {
        await backendRequest('/api/v1/profile', {
          method: 'PUT',
          body: JSON.stringify(profileData),
        });
        notify('Профиль сохранён');
        return true;
      } catch {
        notify('Ошибка сохранения профиля');
        return false;
      }
    },
    [notify]
  );

  const loadBiometrics = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/biometrics');
      if (data && Array.isArray(data.records)) {
        update({ biometrics: data.records });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const addBiometric = useCallback(
    async (metricType, value, deviceType = 'manual') => {
      try {
        await backendRequest('/api/v1/biometrics', {
          method: 'POST',
          body: JSON.stringify({
            metric_type: metricType,
            value,
            timestamp: new Date().toISOString(),
            device_type: deviceType,
          }),
        });
        await loadBiometrics();
        notify('Метрика добавлена');
      } catch {
        notify('Ошибка добавления метрики');
      }
    },
    [loadBiometrics, notify]
  );

  const loadBodyComposition = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/health/body-composition');
      if (data && Array.isArray(data.records)) {
        update({ bodyComposition: data.records });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const saveBodyComposition = useCallback(
    async (record) => {
      try {
        const data = await backendRequest('/api/v1/health/body-composition', {
          method: 'POST',
          body: JSON.stringify(record),
        });
        if (data?.record) {
          const newRecords = [...(state.bodyComposition || []), data.record];
          update({ bodyComposition: newRecords });
        }
        notify('Запись сохранена');
        return true;
      } catch {
        notify('Ошибка сохранения');
        return false;
      }
    },
    [state.bodyComposition, update, notify]
  );

  const loadTrainingPlans = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/training/plans');
      if (Array.isArray(data?.plans)) {
        update({ trainingPlans: data.plans });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const generatePlan = useCallback(
    async (params = {}) => {
      try {
        const data = await backendRequest('/api/v1/training/generate', {
          method: 'POST',
          body: JSON.stringify({
            duration_weeks: params.durationWeeks || 4,
            available_days: params.availableDays || [1, 3, 5],
            class: params.class || 'endurance_basic',
            confidence: params.confidence || 0.8,
          }),
        });
        if (data) {
          await loadTrainingPlans();
          notify('План создан');
          return data;
        }
      } catch {
        notify('Ошибка генерации плана');
      }
      return null;
    },
    [loadTrainingPlans, notify]
  );

  const getPlanDetails = useCallback(
    async (planId) => {
      try {
        const data = await backendRequest(`/api/v1/training/plans/${planId}`);
        return data;
      } catch {
        notify('Ошибка загрузки плана');
        return null;
      }
    },
    [notify]
  );

  const loadProgress = useCallback(async () => {
    try {
      await backendRequest('/api/v1/training/progress');
    } catch {
      // ignore
    }
  }, []);

  const completeWorkout = useCallback(
    async (planId, workoutId, rating = 5, feedback = '') => {
      try {
        await backendRequest('/api/v1/training/complete', {
          method: 'POST',
          body: JSON.stringify({
            plan_id: planId,
            workout_id: workoutId,
            rating,
            feedback,
          }),
        });
      } catch {
        notify('Ошибка завершения тренировки');
      }
    },
    [notify]
  );

  const loadAchievements = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/achievements');
      if (data) {
        const achievements = data.achievements || data.items || [];
        update({ achievementsBackend: achievements });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const loadMeals = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/nutrition/meals');
      if (data && Array.isArray(data.meals)) {
        update({ mealsList: data.meals });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const createMeal = useCallback(
    async (name, calories, time = '') => {
      try {
        const data = await backendRequest('/api/v1/nutrition/meals', {
          method: 'POST',
          body: JSON.stringify({
            name,
            calories,
            time:
              time ||
              new Date().toLocaleTimeString([], {
                hour: '2-digit',
                minute: '2-digit',
              }),
          }),
        });
        if (data?.meal) {
          const newMeals = [...(state.mealsList || []), data.meal];
          update({ mealsList: newMeals });
        }
        notify('Блюдо добавлено');
      } catch {
        notify('Ошибка добавления блюда');
      }
    },
    [state.mealsList, update, notify]
  );

  const removeMeal = useCallback(
    async (mealId) => {
      try {
        await backendRequest(`/api/v1/nutrition/meals/${mealId}`, {
          method: 'DELETE',
        });
        const newMeals = (state.mealsList || []).filter((m) => m.id !== mealId);
        update({ mealsList: newMeals });
        notify('Блюдо удалено');
      } catch {
        notify('Ошибка удаления блюда');
      }
    },
    [state.mealsList, update, notify]
  );

  const loadCalendarEvents = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/calendar/events');
      if (Array.isArray(data?.events)) {
        update({ calendarEventsList: data.events });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const loadVideos = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/videos');
      if (data && Array.isArray(data.videos)) {
        update({ videos: data.videos });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const loadConditions = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/health/conditions');
      if (data && Array.isArray(data.conditions)) {
        update({ conditions: data.conditions });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const createCondition = useCallback(
    async (conditionData) => {
      try {
        const data = await backendRequest('/api/v1/health/conditions', {
          method: 'POST',
          body: JSON.stringify(conditionData),
        });
        if (data?.condition) {
          const newConditions = [...(state.conditions || []), data.condition];
          update({ conditions: newConditions });
        }
        notify('Состояние добавлено');
        return true;
      } catch {
        notify('Ошибка добавления состояния');
        return false;
      }
    },
    [state.conditions, update, notify]
  );

  const deleteCondition = useCallback(
    async (conditionId) => {
      try {
        await backendRequest(`/api/v1/health/conditions/${conditionId}`, {
          method: 'DELETE',
        });
        const newConditions = (state.conditions || []).filter(
          (c) => c.id !== conditionId
        );
        update({ conditions: newConditions });
        notify('Состояние удалено');
      } catch {
        notify('Ошибка удаления состояния');
      }
    },
    [state.conditions, update, notify]
  );

  const loadMenstrualCycles = useCallback(async () => {
    try {
      const data = await backendRequest('/api/v1/health/menstrual-cycles');
      if (Array.isArray(data?.cycles)) {
        update({ menstrualCycles: data.cycles });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const createMenstrualCycle = useCallback(
    async (cycleData) => {
      try {
        const data = await backendRequest('/api/v1/health/menstrual-cycles', {
          method: 'POST',
          body: JSON.stringify(cycleData),
        });
        if (data?.cycle) {
          const newCycles = [...(state.menstrualCycles || []), data.cycle];
          update({ menstrualCycles: newCycles });
        }
        notify('Цикл добавлен');
        return true;
      } catch {
        notify('Ошибка добавления цикла');
        return false;
      }
    },
    [state.menstrualCycles, update, notify]
  );

  const updateMenstrualCycle = useCallback(
    async (cycleId, cycleData) => {
      try {
        const data = await backendRequest(
          `/api/v1/health/menstrual-cycles/${cycleId}`,
          {
            method: 'PUT',
            body: JSON.stringify(cycleData),
          }
        );
        if (data?.cycle) {
          const newCycles = (state.menstrualCycles || []).map((c) =>
            c.id === cycleId ? { ...c, ...data.cycle } : c
          );
          update({ menstrualCycles: newCycles });
        }
        notify('Цикл обновлён');
        return true;
      } catch {
        notify('Ошибка обновления цикла');
        return false;
      }
    },
    [state.menstrualCycles, update, notify]
  );

  const deleteMenstrualCycle = useCallback(
    async (cycleId) => {
      try {
        await backendRequest(`/api/v1/health/menstrual-cycles/${cycleId}`, {
          method: 'DELETE',
        });
        const newCycles = (state.menstrualCycles || []).filter(
          (c) => c.id !== cycleId
        );
        update({ menstrualCycles: newCycles });
        notify('Цикл удалён');
      } catch {
        notify('Ошибка удаления цикла');
      }
    },
    [state.menstrualCycles, update, notify]
  );

  const logWeight = useCallback(
    async (height, weight) => {
      const newHistory = [
        ...state.weightHistory,
        [t('Сегодня', 'Today'), `${weight || state.weight || 70} кг`],
      ];
      update({ weightHistory: newHistory });
      if (height && weight) {
        try {
          await backendRequest('/api/v1/health/body-composition', {
            method: 'POST',
            body: JSON.stringify({
              height_cm: height,
              weight_kg: weight,
              recorded_at: new Date().toISOString(),
            }),
          });
        } catch {
          // ignore
        }
      }
    },
    [state.weightHistory, state.weight, update, t]
  );

  useEffect(() => {
    if (state.registered && !state.profileLoaded) {
      loadProfile();
      loadBiometrics();
      loadTrainingPlans();
      loadMeals();
      loadCalendarEvents();
      loadAchievements();
      loadVideos();
      loadConditions();
      loadMenstrualCycles();
      loadBodyComposition();
    }
  }, [
    state.registered,
    state.profileLoaded,
    loadProfile,
    loadBiometrics,
    loadTrainingPlans,
    loadMeals,
    loadCalendarEvents,
    loadAchievements,
    loadVideos,
    loadConditions,
    loadMenstrualCycles,
    loadBodyComposition,
  ]);

  const value = useMemo(
    () => ({
      state,
      t,
      getMonthName,
      safeText,
      formatRelativeDate,
      notify,
      doLogin,
      doRegister,
      forgotPassword,
      submitResetEmail,
      submitResetCode,
      submitNewPassword,
      continueAsGuest,
      socialLogin,
      logout,
      go,
      openLegal,
      backFromLegal,
      toggleTheme,
      toggleLanguage,
      toggleHighContrast,
      toggleTwoFactor,
      setup2FA,
      confirm2FA,
      verify2FA,
      disable2FA,
      load2FAStatus,
      showNotification,
      drinkWater,
      changeCalendar,
      quickAddEvent,
      setNutritionPeriod,
      setTrainingPeriod,
      openMeal,
      selectMeal,
      saveCustomMeal,
      askAI,
      openMetric,
      closeMetric,
      addWeight,
      toggleWorkout,
      startWorkout,
      editPlace,
      selectPlace,
      addPlace,
      openCalendarPage,
      selectCalendarDay,
      closeDateModal,
      addEventToDay,
      addEvent,
      deleteCalendarEvent,
      sendAI,
      quickAI,
      editProfile,
      changePassword,
      deleteAccount,
      searchFriends,
      showDevices,
      showRestrictions,
      chatSettings,
      saveChatSettings,
      nutritionSettings,
      resumeSurvey,
      scrollStories,
      showAllStories,
      openStory,
      closeStory,
      nextStory,
      prevStory,
      addCustomExercise,
      removeCustomExercise,
      openVideoModal,
      closeVideoModal,
      changeVideoModal,
      connectVideo,
      toggleFriendAdded,
      openFriendProfile,
      showAchievement,
      showAchievementsList,
      surveySelect,
      surveyNext,
      surveyPrev,
      deferSurvey,
      openChat,
      sendChat,
      addStickerToChat,
      uploadPhoto,
      setGoal,
      askAIForDay,
      closeMeal,
      closeAchievement,
      loadProfile,
      saveProfile,
      loadBiometrics,
      addBiometric,
      loadBodyComposition,
      saveBodyComposition,
      loadTrainingPlans,
      generatePlan,
      getPlanDetails,
      loadProgress,
      completeWorkout,
      loadAchievements,
      loadMeals,
      createMeal,
      removeMeal,
      loadCalendarEvents,
      updateEvent,
      loadVideos,
      loadConditions,
      createCondition,
      deleteCondition,
      loadMenstrualCycles,
      createMenstrualCycle,
      updateMenstrualCycle,
      deleteMenstrualCycle,
      logWeight,
    }),
    [
      state,
      t,
      safeText,
      formatRelativeDate,
      notify,
      doLogin,
      doRegister,
      forgotPassword,
      submitResetEmail,
      submitResetCode,
      submitNewPassword,
      continueAsGuest,
      socialLogin,
      go,
      openLegal,
      backFromLegal,
      toggleTheme,
      toggleLanguage,
      toggleHighContrast,
      toggleTwoFactor,
      showNotification,
      drinkWater,
      changeCalendar,
      quickAddEvent,
      setNutritionPeriod,
      setTrainingPeriod,
      openMeal,
      selectMeal,
      saveCustomMeal,
      askAI,
      openMetric,
      closeMetric,
      addWeight,
      toggleWorkout,
      startWorkout,
      editPlace,
      selectPlace,
      addPlace,
      openCalendarPage,
      selectCalendarDay,
      closeDateModal,
      addEventToDay,
      addEvent,
      deleteCalendarEvent,
      sendAI,
      quickAI,
      editProfile,
      changePassword,
      deleteAccount,
      searchFriends,
      showDevices,
      showRestrictions,
      chatSettings,
      saveChatSettings,
      nutritionSettings,
      resumeSurvey,
      scrollStories,
      showAllStories,
      openStory,
      closeStory,
      nextStory,
      prevStory,
      addCustomExercise,
      removeCustomExercise,
      openVideoModal,
      closeVideoModal,
      changeVideoModal,
      connectVideo,
      toggleFriendAdded,
      openFriendProfile,
      showAchievement,
      showAchievementsList,
      surveySelect,
      surveyNext,
      surveyPrev,
      deferSurvey,
      openChat,
      sendChat,
      addStickerToChat,
      uploadPhoto,
      setGoal,
      askAIForDay,
      loadProfile,
      saveProfile,
      loadBiometrics,
      addBiometric,
      loadBodyComposition,
      saveBodyComposition,
      loadTrainingPlans,
      generatePlan,
      getPlanDetails,
      loadProgress,
      completeWorkout,
      loadAchievements,
      loadMeals,
      createMeal,
      removeMeal,
      loadCalendarEvents,
      updateEvent,
      loadVideos,
      loadConditions,
      createCondition,
      deleteCondition,
      loadMenstrualCycles,
      createMenstrualCycle,
      updateMenstrualCycle,
      deleteMenstrualCycle,
      logWeight,
    ]
  );

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp() {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error('useApp must be used within AppProvider');
  return ctx;
}
