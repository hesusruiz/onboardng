import { Routes } from '@angular/router';
import { OnboardingLayoutComponent } from './onboarding/onboarding-layout.component';
import { OnboardingProviderLayoutComponent } from './onboarding/provider/onboarding-provider-layout.component';
import { OnboardingProviderFormalLayoutComponent } from './onboarding/provider-formal/onboarding-provider-formal-layout.component';
import { RegistrationContextGuard } from './onboarding/guards/registration-context.guard';

export const routes: Routes = [
  { path: '', redirectTo: 'register-customer', pathMatch: 'full' },
  { path: 'register-customer', component: OnboardingLayoutComponent },
  { path: 'register-provider', component: OnboardingProviderLayoutComponent },
  { path: 'onboard-provider', component: OnboardingProviderFormalLayoutComponent, canActivate: [RegistrationContextGuard] },
];
