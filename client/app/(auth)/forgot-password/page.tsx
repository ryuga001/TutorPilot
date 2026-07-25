"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";

import {
  useForgotPasswordMutation,
  useResetPasswordMutation,
} from "@/lib/api/authApi";
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
  new_password: z.string().min(8, "At least 8 characters"),
});

type FormValues = z.infer<typeof schema>;

export default function ForgotPasswordPage() {
  const router = useRouter();
  const [sent, setSent] = useState(false);
  const [forgotPassword, forgotState] = useForgotPasswordMutation();
  const [resetPassword, resetState] = useResetPasswordMutation();

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    mode: "onTouched",
    defaultValues: { email: "", otp: "", new_password: "" },
  });

  async function handleSendCode() {
    if (!(await form.trigger("email"))) return;
    try {
      await forgotPassword({ email: form.getValues("email") }).unwrap();
      toast.success("If that email exists, a reset code was sent.");
      setSent(true);
    } catch (err) {
      toast.error(apiErrorMessage(err as never));
    }
  }

  async function onSubmit(values: FormValues) {
    try {
      await resetPassword(values).unwrap();
      toast.success("Password updated. Please sign in.");
      router.replace("/login");
    } catch (err) {
      toast.error(apiErrorMessage(err as never));
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Reset password</CardTitle>
        <CardDescription>
          {sent
            ? "Enter the code and your new password."
            : "We will email you a reset code."}
        </CardDescription>
      </CardHeader>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent className="space-y-4">
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <Input
                      type="email"
                      placeholder="you@company.com"
                      autoComplete="email"
                      disabled={sent}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {sent && (
              <>
                <FormField
                  control={form.control}
                  name="otp"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Reset code</FormLabel>
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
                <FormField
                  control={form.control}
                  name="new_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>New password</FormLabel>
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
            {!sent ? (
              <Button
                type="button"
                className="w-full"
                onClick={handleSendCode}
                disabled={forgotState.isLoading}
              >
                {forgotState.isLoading && <Loader2 className="animate-spin" />}
                Send reset code
              </Button>
            ) : (
              <Button
                type="submit"
                className="w-full"
                disabled={resetState.isLoading}
              >
                {resetState.isLoading && <Loader2 className="animate-spin" />}
                Update password
              </Button>
            )}
            <p className="text-center text-sm text-muted-foreground">
              Remembered it?{" "}
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
