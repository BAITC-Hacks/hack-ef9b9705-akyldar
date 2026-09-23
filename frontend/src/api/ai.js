import {requestWithMeta} from './client.js';

export function getQuestions(description){return requestWithMeta('/api/ai/questions',{method:'POST',body:JSON.stringify({description})})}
export function generateCard(description,answers){return requestWithMeta('/api/ai/card',{method:'POST',body:JSON.stringify({description,answers})})}
