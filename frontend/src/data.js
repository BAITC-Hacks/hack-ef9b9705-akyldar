const base = { title:'',initial_description:'',context:'',need:'',users:'',data:'',constraints:'',expected_result:'',success_criteria:'',contact:'',interaction_format:'',topic:'',rating:0,readiness_level:'draft',confirmed:false,published:false,created_at:'',updated_at:'' };
export const initialTasks = [
  { ...base,id:'sample-1',title:'Помощник по управлению запасами',context:'Региональная торговая сеть управляет запасами на трёх складах с помощью таблиц.',need:'Помочь сотрудникам прогнозировать дефицит и вовремя заказывать товары.',users:'Руководители складов и отдел закупок',data:'Два года данных о продажах, список артикулов и ежедневные выгрузки остатков.',constraints:'Прототип в браузере; использовать только обезличенные данные.',expected_result:'Панель с уведомлениями о дефиците и простым прогнозом спроса.',success_criteria:'Сократить количество дефицитных позиций на 20% за четыре недели.',contact:'innovation@northstar.kz',interaction_format:'Еженедельные онлайн-встречи',topic:'AI & Data',status:'published',scoreOverride:95 },
  { ...base,id:'sample-2',title:'Оптимизация маршрутов доставки',context:'Наши курьеры теряют время из-за неэффективных маршрутов.',need:'Планировать ежедневные маршруты городских курьеров.',users:'Диспетчеры и курьеры',data:'Обезличенные координаты доставок и временные окна.',constraints:'Прототип для одного города.',expected_result:'Карта с оптимизированными маршрутами на день.',success_criteria:'Сократить общее время в пути на 15%.',contact:'hello@flow.kz',topic:'Logistics',status:'published',scoreOverride:82 },
  { ...base,id:'sample-3',title:'Анализ отзывов клиентов',context:'Наша команда каждую неделю изучает сотни комментариев клиентов.',need:'Находить повторяющиеся проблемы в отзывах.',users:'Руководители службы поддержки',constraints:'Не раскрывать персональные данные клиентов.',expected_result:'Сводная панель с основными темами.',contact:'team@voice.kz',topic:'AI & Data',status:'published',scoreOverride:67 },
  { ...base,id:'sample-4',title:'Очередь в кампусном кафе',context:'Студенты слишком долго ждут обед.',need:'Найти способы сократить очереди.',users:'Сотрудники кафе и студенты',topic:'Food & Retail',status:'published',scoreOverride:48 },
  { ...base,id:'sample-5',title:'Удобный сайт',need:'Нам нужен новый сайт.',topic:'Technology',status:'published',scoreOverride:25 }
];
export const fields = [
  ['title','Название задачи','text'],['context','Контекст','textarea'],['need','Потребность','textarea'],['users','Пользователи','text'],['data','Данные и материалы','textarea'],['constraints','Ограничения','textarea'],['expected_result','Ожидаемый результат','textarea'],['success_criteria','Критерии успеха','textarea'],['contact','Контакт','text'],['interaction_format','Формат взаимодействия','text'],['topic','Тема','text']
];
export const breakdown = [
  ['Контекст и потребность',20,'context_need'],['Данные и материалы',20,'data'],['Ожидаемый результат',15,'expected_result'],['Критерии успеха',15,'success_criteria'],['Ограничения',10,'constraints'],['Пользователи',10,'users'],['Связь с бизнесом',10,'business_connection']
];
export function rating(task){
  if(task.scoreOverride != null) return task.scoreOverride;
  if(task.rating != null && Number.isFinite(Number(task.rating))) return Number(task.rating);
  return calculateMockRating(task).score;
}
export function readiness(score){return score<40?'draft':score<70?'working':score<90?'ready':'priority'}
const readinessLabels={draft:'Черновик',working:'Рабочая',ready:'Готовая',priority:'Приоритетная'};
const topicLabels={'All topics':'Все темы','AI & Data':'ИИ и данные','Logistics':'Логистика','Food & Retail':'Еда и розничная торговля','Technology':'Технологии',logistics:'Логистика',education:'Образование',retail:'Розничная торговля',analytics:'Аналитика','customer-service':'Клиентский сервис'};
const fieldLabels={context:'Контекст',need:'Потребность',users:'Пользователи',data:'Данные и материалы',constraints:'Ограничения',expected_result:'Ожидаемый результат',success_criteria:'Критерии успеха',contact:'Контакт',interaction_format:'Формат взаимодействия'};
const proposalStatusLabels={pending:'На рассмотрении',accepted:'Принято',rejected:'Отклонено'};
export function readinessLabel(level){const key=String(level||'draft').toLowerCase();return readinessLabels[key]||String(level||'Черновик')}
export function topicLabel(topic){return topicLabels[topic]||topic||'Без темы'}
export function fieldLabel(key){return fieldLabels[key]||key}
export function proposalStatusLabel(status){return proposalStatusLabels[String(status||'').toLowerCase()]||status||'На рассмотрении'}
export const questions=['Кто будет пользоваться решением каждый день?','Какие данные или материалы вы можете предоставить студенческой команде?','Как вы поймёте, что решение достигло успеха?'];
export const fallbackQuestions=questions;
export function calculateMockRating(task={}){
  const has=key=>Boolean(String(task[key]||'').trim());
  const values={context_need:has('context')&&has('need')?20:0,data:has('data')?20:0,expected_result:has('expected_result')?15:0,success_criteria:has('success_criteria')?15:0,constraints:has('constraints')?10:0,users:has('users')?10:0,business_connection:(has('contact')?5:0)+(has('interaction_format')?5:0)};
  const missing=['context','need','data','expected_result','success_criteria','constraints','users','contact','interaction_format'].filter(hasKey=>!has(hasKey));
  const score=Object.values(values).reduce((total,value)=>total+value,0);
  return {score,level:readiness(score),breakdown:values,missing};
}
export function mockCard(description,answers){return {title:'',context:String(description||'').trim(),need:'',users:answers?.[0]?.answer?.trim()||'',data:answers?.[1]?.answer?.trim()||'',constraints:'',expected_result:'',success_criteria:answers?.[2]?.answer?.trim()||'',contact:'',interaction_format:'',topic:''};}
export function mockTask(description,topic=''){
  const now=new Date().toISOString();
  const task={...base,id:crypto.randomUUID(),initial_description:String(description||'').trim(),topic:String(topic||'').trim(),created_at:now,updated_at:now};
  const score=calculateMockRating(task);
  return {...task,...score,rating:score.score,readiness_level:score.level};
}
export function freshTask(description,answers){const task={...mockTask(description),...mockCard(description,answers)};const score=calculateMockRating(task);return {...task,...score,rating:score.score,readiness_level:score.level};}
