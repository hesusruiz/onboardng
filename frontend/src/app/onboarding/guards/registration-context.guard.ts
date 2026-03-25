import { Injectable, inject } from '@angular/core';
import { Router } from '@angular/router';
import { RegistrationContextService } from '../services/registration-context.service';

@Injectable({
  providedIn: 'root'
})
export class RegistrationContextGuard {
  private registrationContext = inject(RegistrationContextService);
  private router = inject(Router);

  canActivate(): boolean {
    const context = this.registrationContext.getContext();

    // Check if all required context fields are populated
    if (context.vatId && context.email && context.code) {
      return true;
    }

    // Redirect to provider registration if context is missing
    this.router.navigate(['/register-provider']);
    return false;
  }
}
