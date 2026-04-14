import { Component, ViewEncapsulation, Input, ElementRef, AfterViewInit, OnDestroy, HostBinding } from '@angular/core';

@Component({
  selector: 'app-progress-bar',
  standalone: true,
  template: `
    <div class="flex items-center justify-between mb-2">
      <span class="text-sm text-gray-500">Step {{ currentStep }} of {{ totalSteps }}</span>
      <span class="text-base font-semibold leading-6 tracking-[0.5px] text-[#111827]">{{ stepTitle }}</span>
    </div>
    <div class="w-full bg-gray-200 rounded-full h-1.5">
      <div
        class="bg-[#339988] h-1.5 rounded-full transition-all duration-300"
        [style.width.%]="(currentStep / totalSteps) * 100">
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
    :host:not(.plain) {
      position: sticky;
      top: 0;
      z-index: 10;
      background-color: rgb(249 250 251);
      padding: 2rem 0 1rem;
      border-bottom: 1px solid transparent;
      transition: border-color 0.15s;
    }
  `]
})
export class ProgressBarComponent implements AfterViewInit, OnDestroy {
  @Input() currentStep = 1;
  @Input() totalSteps = 3;
  @Input() stepTitle = '';
  @Input() plain = false;

  @HostBinding('class.plain')
  get isPlain(): boolean { return this.plain; }

  @HostBinding('style.border-bottom-color')
  borderColor = 'transparent';

  private scrollParent: HTMLElement | null = null;
  private scrollHandler = () => {
    if (this.scrollParent) {
      this.borderColor = this.scrollParent.scrollTop > 0 ? '#B6CAEC' : 'transparent';
    }
  };

  constructor(private el: ElementRef) {}

  ngAfterViewInit(): void {
    if (this.plain) return;
    this.scrollParent = this.findScrollParent(this.el.nativeElement);
    if (this.scrollParent) {
      this.scrollParent.addEventListener('scroll', this.scrollHandler, { passive: true });
    }
  }

  ngOnDestroy(): void {
    if (this.scrollParent) {
      this.scrollParent.removeEventListener('scroll', this.scrollHandler);
    }
  }

  private findScrollParent(el: HTMLElement): HTMLElement | null {
    let parent = el.parentElement;
    while (parent) {
      const style = getComputedStyle(parent);
      if (style.overflowY === 'auto' || style.overflowY === 'scroll') {
        return parent;
      }
      parent = parent.parentElement;
    }
    return null;
  }
}
