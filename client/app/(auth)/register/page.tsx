"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { ArrowLeft, Loader2 } from "lucide-react";

import {
  useRegisterMutation,
  useSendVerificationMutation,
  useVerifyEmailMutation,
} from "@/lib/api/authApi";
import { useAppDispatch } from "@/lib/hooks";
import { setCredentials } from "@/lib/features/authSlice";
import { apiErrorMessage } from "@/lib/api-error";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";

const schema = z.object({
  email: z.string().email("Enter a valid email"),
  otp: z.string().length(6, "Enter the 6-digit code"),
  org_name: z.string().min(2, "Organization name is too short"),
  first_name: z.string().min(1, "Required"),
  last_name: z.string().min(1, "Required"),
  password: z.string().min(8, "At least 8 characters"),
});

type FormValues = z.infer<typeof schema>;

type Step = 1 | 2 | 3;

const STEP_META: Record<Step, { title: string; description: string }> = {
  1: { title: "Verify email", description: "We will send you a 6-digit code." },
  2: { title: "Enter code", description: "Check your inbox for the code." },
  3: {
    title: "Organization details",
    description: "Set up your workspace and admin account.",
  },
};

export default function RegisterPage() {
  const router = useRouter();
  const dispatch = useAppDispatch();
  const [step, setStep] = useState<Step>(1);

  const [sendVerification, sendState] = useSendVerificationMutation();
  const [verifyEmail, verifyState] = useVerifyEmailMutation();
  const [register, registerState] = useRegisterMutation();

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    mode: "onTouched",
    defaultValues: {
      email: "",
      otp: "",
      org_name: "",
      first_name: "",
      last_name: "",
      password: "",
    },
  });

  async function handleSendCode() {
    if (!(await form.trigger("email"))) return;
    try {
      await sendVerification({ email: form.getValues("email") }).unwrap();
      toast.success("Verification code sent.");
      setStep(2);
    } catch (err) {
      toast.error(apiErrorMessage(err as never));
    }
  }

  async function handleVerifyCode() {
    if (!(await form.trigger("otp"))) return;
    try {
      await verifyEmail({
        email: form.getValues("email"),
        otp: form.getValues("otp"),
      }).unwrap();
      toast.success("Email verified.");
      setStep(3);
    } catch (err) {
      toast.error(apiErrorMessage(err as never));
    }
  }

  async function onSubmit(values: FormValues) {
    try {
      const res = await register({
        org_name: values.org_name,
        first_name: values.first_name,
        last_name: values.last_name,
        email: values.email,
        password: values.password,
      }).unwrap();
      dispatch(setCredentials(res));
      toast.success("Organization created!");
      router.replace("/dashboard");
    } catch (err) {
      toast.error(apiErrorMessage(err as never));
    }
  }

  const meta = STEP_META[step];

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          {step > 1 && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => setStep((s) => (s - 1) as Step)}
            >
              <ArrowLeft />
            </Button>
          )}
          <div>
            <CardTitle>{meta.title}</CardTitle>
            <CardDescription>{meta.description}</CardDescription>
          </div>
        </div>
        <p className="pt-2 text-xs font-medium text-muted-foreground">
          Step {step} of 3
        </p>
      </CardHeader>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent className="space-y-4">
            {step === 1 && (
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Work email</FormLabel>
                    <FormControl>
                      <Input
                        type="email"
                        placeholder="you@company.com"
                        autoComplete="email"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            {step === 2 && (
              <FormField
                control={form.control}
                name="otp"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>6-digit code</FormLabel>
                    <FormControl>
                      <Input
                        inputMode="numeric"
                        maxLength={6}
                        placeholder="123456"
                        className="tracking-[0.5em]"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            {step === 3 && (
              <>
                <FormField
                  control={form.control}
                  name="org_name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Organization name</FormLabel>
                      <FormControl>
                        <Input placeholder="Acme Tutoring" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <div className="grid grid-cols-2 gap-3">
                  <FormField
                    control={form.control}
                    name="first_name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>First name</FormLabel>
                        <FormControl>
                          <Input placeholder="Ada" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="last_name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Last name</FormLabel>
                        <FormControl>
                          <Input placeholder="Lovelace" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
                <FormField
                  control={form.control}
                  name="password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Password</FormLabel>
                      <FormControl>
                        <Input
                          type="password"
                          placeholder="••••••••"
                          autoComplete="new-password"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </>
            )}
          </CardContent>

          <CardFooter className="flex flex-col gap-4">
            {step === 1 && (
              <Button
                type="button"
                className="w-full"
                onClick={handleSendCode}
                disabled={sendState.isLoading}
              >
                {sendState.isLoading && <Loader2 className="animate-spin" />}
                Send code
              </Button>
            )}
            {step === 2 && (
              <div className="flex w-full flex-col gap-2">
                <Button
                  type="button"
                  className="w-full"
                  onClick={handleVerifyCode}
                  disabled={verifyState.isLoading}
                >
                  {verifyState.isLoading && (
                    <Loader2 className="animate-spin" />
                  )}
                  Verify code
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  className="w-full"
                  onClick={handleSendCode}
                  disabled={sendState.isLoading}
                >
                  Resend code
                </Button>
              </div>
            )}
            {step === 3 && (
              <Button
                type="submit"
                className="w-full"
                disabled={registerState.isLoading}
              >
                {registerState.isLoading && (
                  <Loader2 className="animate-spin" />
                )}
                Create organization
              </Button>
            )}
            <p className="text-center text-sm text-muted-foreground">
              Already have an account?{" "}
              <Link href="/login" className="text-primary hover:underline">
                Sign in
              </Link>
            </p>
          </CardFooter>
        </form>
      </Form>
    </Card>
  );
}
