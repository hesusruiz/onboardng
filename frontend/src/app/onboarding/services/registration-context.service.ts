import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

export interface RegistrationContext {
  vatId: string;
  email: string;
  code: string;
}

@Injectable({
  providedIn: 'root'
})
export class RegistrationContextService {
  private context = new BehaviorSubject<RegistrationContext>({
    vatId: '',
    email: '',
    code: ''
  });

  context$ = this.context.asObservable();

  setContext(context: Partial<RegistrationContext>) {
    this.context.next({
      ...this.context.value,
      ...context
    });
  }

  getContext(): RegistrationContext {
    return this.context.value;
  }

  clearContext() {
    this.context.next({
      vatId: '',
      email: '',
      code: ''
    });
  }
}
