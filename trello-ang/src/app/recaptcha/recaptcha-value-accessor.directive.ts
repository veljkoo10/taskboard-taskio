import { Directive, forwardRef, ElementRef, Renderer2, OnInit } from '@angular/core';
import { NG_VALUE_ACCESSOR, ControlValueAccessor } from '@angular/forms';

@Directive({
  selector: 're-captcha',
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => RecaptchaValueAccessor),
      multi: true,
    },
  ],
})
export class RecaptchaValueAccessor implements ControlValueAccessor, OnInit {
  private onChange: (value: any) => void = () => {};  // Provide a default empty function
  private onTouched: () => void = () => {};           // Provide a default empty function

  constructor(private el: ElementRef, private renderer: Renderer2) {}

  ngOnInit() {
    const captchaElement = this.el.nativeElement;

    // Listen for reCAPTCHA verification
    captchaElement.addEventListener('verify', (event: CustomEvent) => {
      this.onChange(event.detail.response);
    });
  }

  writeValue(value: any): void {
    const captchaElement = this.el.nativeElement;
    if (value) {
      // Reset reCAPTCHA state
      captchaElement.reset();
    }
  }

  registerOnChange(fn: (value: any) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }
}
