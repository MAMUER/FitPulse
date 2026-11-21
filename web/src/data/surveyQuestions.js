// Данные онбординг-опроса. Вынесены из Survey.jsx: это чистая декларативная
// таблица (вопросы и варианты ответов), а не логика экрана. Раньше массив
// занимал ~112 строк внутри компонента, и Sonar видел в нём «дублирование»
// из-за многократно повторяющихся пар [ru, en].
export const QUESTIONS = [
  {
    title: ['Аллергии', 'Allergies'],
    key: 'allergies',
    multi: true,
    options: [
      ['Орехи', 'Nuts'],
      ['Молоко', 'Milk'],
      ['Яйца', 'Eggs'],
      ['Глютен', 'Gluten'],
      ['Морепродукты', 'Seafood'],
      ['Соя', 'Soy'],
      ['Нет аллергий', 'No allergies'],
    ],
  },
  {
    title: ['Ограничения по здоровью', 'Health restrictions'],
    key: 'restrictions',
    multi: true,
    options: [
      ['Диабет', 'Diabetes'],
      ['Гипертония', 'Hypertension'],
      ['Сердечные', 'Heart conditions'],
      ['Почечные', 'Kidney conditions'],
      ['Печёночные', 'Liver conditions'],
      ['Ортопедические', 'Orthopedic conditions'],
      ['Нет ограничений', 'No restrictions'],
    ],
  },
  {
    title: ['Цели занятий', 'Fitness goals'],
    key: 'goals',
    multi: false,
    options: [
      ['Похудеть', 'Lose weight'],
      ['Набрать мышечную массу', 'Build muscle'],
      ['Поддерживать форму', 'Stay fit'],
      ['Повысить выносливость', 'Improve endurance'],
      ['Улучшить здоровье', 'Improve health'],
      ['Снизить стресс', 'Reduce stress'],
    ],
  },
  {
    title: ['Предпочтения в еде', 'Diet preferences'],
    key: 'diet',
    options: [
      ['Обычное', 'Regular'],
      ['Вегетарианское', 'Vegetarian'],
      ['Веганское', 'Vegan'],
      ['Кето', 'Keto'],
      ['Безглютеновое', 'Gluten-free'],
      ['Средиземноморское', 'Mediterranean'],
    ],
  },
  {
    title: ['Уровень активности', 'Activity level'],
    key: 'activity',
    options: [
      ['Сидячий', 'Sedentary'],
      ['Лёгкий (1--2 раза/нед)', 'Light (1--2 times/week)'],
      ['Средний (3--4 раза/нед)', 'Moderate (3--4 times/week)'],
      ['Высокий (5--6 раз/нед)', 'High (5--6 times/week)'],
      ['Профессиональный спорт', 'Professional sports'],
    ],
  },
  {
    title: ['Сон (часов)', 'Sleep (hours)'],
    key: 'sleepHours',
    options: [
      ['Менее 6 ч', 'Under 6 hours'],
      ['6--7 ч', '6--7 hours'],
      ['7--8 ч', '7--8 hours'],
      ['8--9 ч', '8--9 hours'],
      ['Более 9 ч', 'Over 9 hours'],
    ],
  },
  {
    title: ['Питьевой режим', 'Water intake'],
    key: 'waterIntake',
    options: [
      ['Менее 1 л', 'Under 1 L'],
      ['1--1.5 л', '1--1.5 L'],
      ['1.5--2 л', '1.5--2 L'],
      ['Более 2 л', 'Over 2 L'],
    ],
  },
  {
    title: ['Предпочтения по тренировкам', 'Preferred workouts'],
    key: 'preferredWorkouts',
    multi: true,
    options: [
      ['Силовые', 'Strength training'],
      ['Кардио', 'Cardio'],
      ['Йога', 'Yoga'],
      ['Пилатес', 'Pilates'],
      ['Плавание', 'Swimming'],
      ['Бег', 'Running'],
      ['Функциональные', 'Functional training'],
    ],
  },
  {
    title: ['Дни тренировок в неделю', 'Training days per week'],
    key: 'trainingDays',
    options: [
      ['1--2 дня', '1--2 days'],
      ['3 дня', '3 days'],
      ['4 дня', '4 days'],
      ['5 дней', '5 days'],
      ['6--7 дней', '6--7 days'],
    ],
  },
];
