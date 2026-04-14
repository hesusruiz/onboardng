import { Component, ViewEncapsulation, EventEmitter, Input, Output } from '@angular/core';
import { NgClass } from '@angular/common';

@Component({
  selector: 'app-step-footer',
  standalone: true,
  imports: [NgClass],
  template: `
    <footer class="fixed bottom-0 left-0 w-full bg-white border-t border-gray-200 px-8 py-4 z-40">
      <div class="max-w-3xl mx-auto flex items-center" [ngClass]="showBack ? 'justify-between' : 'justify-end'">
        @if (showBack) {
          <button
            type="button"
            class="text-sm font-medium text-primary-100 hover:underline cursor-pointer"
            (click)="back.emit()">
            Back
          </button>
        }
        <button
          type="button"
          class="px-5 py-3 text-sm font-medium text-white rounded-md bg-primary-100 hover:bg-primary-50 transition-colors"
          [ngClass]="nextDisabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'"
          [disabled]="nextDisabled"
          (click)="next.emit()">
          {{ nextLabel }}
        </button>
      </div>
    </footer>
  `
})
export class StepFooterComponent {
  @Input() showBack = true;
  @Input() nextLabel = 'Continue';
  @Input() nextDisabled = false;

  @Output() back = new EventEmitter<void>();
  @Output() next = new EventEmitter<void>();
}
