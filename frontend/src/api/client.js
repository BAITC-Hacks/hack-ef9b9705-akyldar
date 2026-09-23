const API_URL=(import.meta.env.VITE_API_URL||'http://localhost:8080').replace(/\/$/,'');

export class ApiError extends Error{
  constructor(message,status=0){super(message);this.name='ApiError';this.status=status;}
}

export async function request(path,options={}){
  const init={...options,headers:{...(options.body?{'Content-Type':'application/json'}:{}),...(options.headers||{})}};
  try{
    const response=await fetch(`${API_URL}${path}`,init);
    const text=await response.text();
    let body=null;
    try{body=text?JSON.parse(text):null}catch{body=null}
    if(!response.ok){
      const rawMessage=typeof body?.error==='string'?body.error:body?.error?.message;
      throw new ApiError(frontendErrorMessage(rawMessage),response.status);
    }
    return body;
  }catch(error){
    if(error instanceof ApiError) throw error;
    throw new ApiError(`Не удаётся подключиться к серверу по адресу ${API_URL}. Убедитесь, что сервер запущен.`);
  }
}

export function apiErrorMessage(error){
  return error instanceof ApiError?error.message:'Произошла ошибка. Попробуйте ещё раз.';
}

const errorTranslations={
  'initial_description is required':'Введите описание задачи.',
  'invalid request body':'Некорректное содержимое запроса.',
  'invalid sort parameter':'Некорректный параметр сортировки.',
  'invalid readiness level':'Некорректный уровень готовности.',
  'invalid task id':'Некорректный идентификатор задачи.',
  'task not found':'Задача не найдена.',
  'task must be confirmed before publishing':'Перед публикацией подтвердите задачу.',
  'task must be published before accepting proposals':'Предложения можно отправлять только для опубликованной задачи.',
  'team not found':'Команда не найдена.',
  'team_id is required':'Укажите идентификатор команды.',
  'team_id must be positive':'Идентификатор команды должен быть положительным.',
  'idea is required':'Опишите идею предложения.',
  'plan is required':'Укажите план действий.',
  'deadline is required':'Укажите срок выполнения.',
  'prototype_url is required':'Укажите ссылку на прототип.',
  'invalid proposal id':'Некорректный идентификатор предложения.',
  'invalid proposal status':'Некорректный статус предложения.',
  'proposal not found':'Предложение не найдено.',
  'method not allowed':'Этот способ действия недоступен.',
  'internal server error':'Внутренняя ошибка сервера.',
  'The request could not be completed.':'Не удалось выполнить запрос.',
  'The request could not be processed.':'Не удалось обработать запрос.',
};

function frontendErrorMessage(message){
  return errorTranslations[message]||'Не удалось выполнить запрос.';
}
