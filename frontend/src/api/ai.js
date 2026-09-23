import {request} from './client.js';

export function getQuestions(description){return request('/api/ai/questions',{method:'POST',body:JSON.stringify({description})})}
export function generateCard(description,answers){return request('/api/ai/card',{method:'POST',body:JSON.stringify({description,answers})})}
