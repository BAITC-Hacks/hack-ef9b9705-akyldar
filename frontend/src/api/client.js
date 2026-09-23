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
    if(!response.ok) throw new ApiError(body?.error||'The request could not be completed.',response.status);
    return body;
  }catch(error){
    if(error instanceof ApiError) throw error;
    throw new ApiError(`Cannot connect to the backend at ${API_URL}. Make sure it is running.`);
  }
}

export function apiErrorMessage(error){
  return error instanceof ApiError?error.message:'Something went wrong. Please try again.';
}
