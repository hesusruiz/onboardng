import { Component, ViewEncapsulation } from '@angular/core';
import { OnboardingProviderFormalComponent } from './onboarding-provider-formal.component';

@Component({
  selector: 'app-onboarding-provider-formal-layout',
  standalone: true,
  imports: [OnboardingProviderFormalComponent],
  templateUrl: './onboarding-provider-formal-layout.component.html',
  styleUrl: './onboarding-provider-formal-layout.component.css',
  encapsulation: ViewEncapsulation.None
})
export class OnboardingProviderFormalLayoutComponent {}
