import {request} from './client.js';

const editableFields=['title','context','need','users','data','constraints','expected_result','success_criteria','contact','interaction_format','topic'];

export function editableTask(task){
  return editableFields.reduce((payload,key)=>{payload[key]=task?.[key]||'';return payload},{});
}

export function createTask({initial_description,topic}){
  return request('/api/tasks',{method:'POST',body:JSON.stringify({initial_description,topic})});
}

export function getTask(id){return request(`/api/tasks/${id}`)}

export function listTasks({topic='',level='',sort=''}={}){
  const params=new URLSearchParams();
  if(topic) params.set('topic',topic);
  if(level) params.set('level',level);
  if(sort) params.set('sort',sort);
  const query=params.toString();
  return request(`/api/tasks${query?`?${query}`:''}`);
}

export function updateTask(id,task){return request(`/api/tasks/${id}`,{method:'PUT',body:JSON.stringify(editableTask(task))})}
export function getTaskRating(id){return request(`/api/tasks/${id}/rating`)}
export function confirmTask(id){return request(`/api/tasks/${id}/confirm`,{method:'POST'})}
export function publishTask(id){return request(`/api/tasks/${id}/publish`,{method:'POST'})}
