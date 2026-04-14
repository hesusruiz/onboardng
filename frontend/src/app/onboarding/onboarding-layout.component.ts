import { Component, ViewEncapsulation } from '@angular/core';
import { OnboardingComponent } from './onboarding.component';

@Component({
  selector: 'app-onboarding-layout',
  standalone: true,
  imports: [OnboardingComponent],
  templateUrl: './onboarding-layout.component.html',
  styleUrl: './onboarding-layout.component.css',
  encapsulation: ViewEncapsulation.None
})
export class OnboardingLayoutComponent {}
