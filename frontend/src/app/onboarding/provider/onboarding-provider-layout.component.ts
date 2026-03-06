import { Component, ViewEncapsulation } from '@angular/core';
import { OnboardingProviderComponent } from './onboarding-provider.component';

@Component({
  selector: 'app-onboarding-provider-layout',
  standalone: true,
  imports: [OnboardingProviderComponent],
  templateUrl: './onboarding-provider-layout.component.html',
  styleUrl: './onboarding-provider-layout.component.css',
  encapsulation: ViewEncapsulation.None
})
export class OnboardingProviderLayoutComponent {}
